# mytun Web 面板 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 mytun 加 Web 面板：Gin 提供 5 个 API（status/tun 创建删除/send/logs），4 个 WebComponent 展示操作，单 TUN。

**Architecture:** 新增 `internal/tunmgr`（mutex 单实例状态机 + stats + 环形 log）隔离状态；`internal/web` 放 Gin 路由与 handler（只做绑定校验+委托+信封）；前端 `web/src/*.ts` 经 esbuild 打包后 `go:embed` 进二进制。

**Tech Stack:** Go 1.27, Gin, wireguard-go/tun（复用）, TypeScript WebComponents, vitest + happy-dom, esbuild

## Global Constraints

- TDD：先 RED 后 GREEN，禁止事后补测试（AGENTS.md §1）。
- 统一信封 `{ok:true,data}` / `{ok:false,error:{code,message}}`；bad json→400、无 tun→404、重复创建→409。
- 单测零网络、零 root：tun 建卡只 mock/fake，需 root 走 `scripts/darwin_e2e.sh`。
- 无状态函数优先：handler 只做绑定+委托+信封，判定下沉。
- 前端工具链全局安装，不写 `web/package.json`、不提交 `web/node_modules`；构建产物进 `build/`（gitignored）。
- `make test-all`（go test + vitest）全绿才可提交。

---

### Task 1: tunmgr 状态机 + stats + 环形日志

**Files:**
- Create: `internal/tunmgr/manager.go`
- Test: `internal/tunmgr/manager_test.go`

**Interfaces:**
- Consumes: `internal/tun.TUN`（已有 Create/Read/Write/ActualName/Close），`internal/netconf.ConfigAddr`，`internal/echo.ReplyToPing`。
- Produces（Task 3/4 使用）:
  - `type State string` — `"down" | "up"`
  - `type Stats struct { RxPackets, TxPackets, RxBytes, TxBytes uint64 }`
  - `type LogEntry struct { Time string; Dir string; Summary string }`
  - `type Manager struct { ... }`
  - `func New(dev Device, configurer func(ifname string) error) *Manager`
  - `func (m *Manager) Create(addr string, mtu int) (ifname string, err error)` — 已存在返回 `ErrExists`
  - `func (m *Manager) Delete() error` — 不存在返回 `ErrNoTun`
  - `func (m *Manager) Status() (state State, ifname, addr string, mtu int, s Stats)`
  - `func (m *Manager) PushLog(dir, summary string)` / `func (m *Manager) Logs(limit int) []LogEntry`（环形 200 条）
  - `func (m *Manager) AddRx(n int)` / `func (m *Manager) AddTx(n int)`
  - `var ErrExists, ErrNoTun error`
  - `Device` 接口：`Read(p []byte) (int,error)` `Write(p []byte)(int,error)` `ActualName()(string,error)` `Close() error`（fake device 实现此接口注入）

- [ ] **Step 1: 写 failing test** — `internal/tunmgr/manager_test.go`：fake device；Create 后 Status 为 up；重复 Create 返回 ErrExists；Delete 后为 down；重复 Delete 返回 ErrNoTun；PushLog 250 条后 Logs(300) 只剩 200 条；AddRx/AddTx 计数正确。
- [ ] **Step 2: 运行确认失败** — `go test ./internal/tunmgr/ -run TestManager -v`，预期 FAIL（包不存在）。
- [ ] **Step 3: 最小实现** — `manager.go`：mutex + 状态 + stats + 切片环形 log（超 200 从头截断）。
- [ ] **Step 4: 运行确认通过** — `go test ./internal/tunmgr/ -v`，预期 PASS。
- [ ] **Step 5: Commit** — `git add internal/tunmgr && git commit -m "feat: add tunmgr state machine with stats and ring log"`。

---

### Task 2: send 包构造纯函数

**Files:**
- Create: `internal/packet/packet.go`
- Test: `internal/packet/packet_test.go`

**Interfaces:**
- Consumes: 无（纯函数；校验和复用 `internal/echo.Checksum`）。
- Produces（Task 3 `POST /api/send` 使用）:
  - `func BuildIPv4EchoRequest(src, dst netip.Addr, size int) ([]byte, error)` — size 为总长度（含 20 字节头），范围 28~65535；payload 零填充；返回可直接 Write 的完整 IP 包。
  - `func ParseSummary(pkt []byte) string` — `"src->dst proto=N len=N"`，短包/非 IPv4 返回 `"short len=N"` / `"non-ipv4 ver=N"`（给日志用）。

- [ ] **Step 1: 写 failing test** — echo request 头字段断言（ver=4、proto=1、type=8、src/dst 正确、总长=size、IP 校验和自洽为 0）；非法 size 返回 error；ParseSummary 覆盖短包/TCP/正常包。
- [ ] **Step 2: 运行确认失败** — `go test ./internal/packet/ -v`，预期 FAIL。
- [ ] **Step 3: 最小实现** — 组 IP 头 + ICMP 头 + 零填充，调 `echo.Checksum` 填两处校验和。
- [ ] **Step 4: 运行确认通过** — `go test ./internal/packet/ -v`，预期 PASS。
- [ ] **Step 5: Commit** — `git add internal/packet && git commit -m "feat: add packet build and summary helpers"`。

---

### Task 3: Gin API（5 接口 + 信封）

**Files:**
- Create: `internal/web/server.go` — `func NewRouter(m *tunmgr.Manager, version string) *gin.Engine`，路由：`GET /healthz`，`GET /api/status`，`POST /api/tun`，`DELETE /api/tun`，`POST /api/send`，`GET /api/logs`；`/` 返回内嵌 index.html（Task 5 嵌入；本任务先返回占位 HTML 字符串常量，Task 5 替换为 embed）。
- Create: `internal/web/envelope.go` — `func ok(c *gin.Context, data any)` / `func fail(c *gin.Context, status int, code, msg string)`。
- Test: `internal/web/server_test.go` — gin TestMode + httptest + fake tunmgr：healthz 200；无 tun 时 status.state=down；POST /api/tun 非法 addr →400；正常创建→200 且 status up；重复创建→409；DELETE→200 后 status down；重复删→404；POST /api/send 无 tun→404，有 tun→200；GET /api/logs limit 生效。
- Modify: `go.mod` — 新增 `github.com/gin-gonic/gin`。

**Interfaces:**
- Consumes: Task 1 `tunmgr.Manager` 方法全集，Task 2 `packet.BuildIPv4EchoRequest`。
- Produces（Task 4 使用）: `NewRouter` 签名锁定。

- [ ] **Step 1: 写 failing test** — 上述全部用例先写（约 10 个子测试）。
- [ ] **Step 2: 运行确认失败** — `go test ./internal/web/ -v`，预期 FAIL（包不存在）。
- [ ] **Step 3: 最小实现** — envelope + 6 路由；send 逻辑：查 tun device（manager 需暴露 `Device() Device` 方法——若 Task 1 没加，本任务补上并补测试）→ Build 包 → Write → AddTx + PushLog。
- [ ] **Step 4: 运行确认通过** — `go test ./internal/web/ -v`，预期全 PASS。
- [ ] **Step 5: Commit** — `git add internal/web go.mod go.sum && git commit -m "feat: add gin api with envelope and handler tests"`。

---

### Task 4: cmd 接线（--web flag + 收包循环记账）

**Files:**
- Modify: `cmd/mytun/main.go` — 加 `--web` flag（默认 `:8080`，空字符串关闭）；创建 tunmgr.Manager 替代直调 tun；收包循环每包调 `AddRx + PushLog(ParseSummary)`；echo 回包成功调 `AddTx`。

**Interfaces:**
- Consumes: Task 1 Manager，Task 3 `web.NewRouter`。
- Produces: 无（终端接线）。

- [ ] **Step 1: 改代码** — flag + manager 接线（无新单测；已有单测回归）。
- [ ] **Step 2: 运行回归** — `go test ./...` 全 PASS；`go build ./...` 通过。
- [ ] **Step 3: Commit** — `git add cmd/mytun && git commit -m "feat: wire web server and stats into main"`。

---

### Task 5: 前端组件 + 打包嵌入

**Files:**
- Create: `web/src/api.ts`（抄 ryba 信封解析）、`web/src/app.ts`、`web/src/{tun-status,tun-controls,tun-send,tun-log}.ts`、`web/index.html`、`web/style.css`、`web/tsconfig.json`、`web/vitest.config.mjs`、`scripts/build_web.cjs`
- Test: `web/tests/{api,tun-status,tun-controls,tun-send,tun-log}.test.ts`
- Modify: `internal/web/server.go` — `/` 改为 `go:embed` 打包后的 `build/web-dist/mytun-web.js` + index.html（dev 模式 `--web-root` 读文件，抄 ryba）。
- Modify: `Makefile` — 新增 `web-build`（esbuild）、`test-front`（vitest run）、`test-all`（go test + vitest）；`build` 依赖 `web-build`。
- Modify: `.gitignore` — 加 `build/`。

- [ ] **Step 1: 写 failing test** — 5 个 test.ts：api 信封 ok/err 解析；每个组件 happy-dom 挂载后含关键文本/按钮；send 组件点击后 fetch 被调（mock fetch）。
- [ ] **Step 2: 运行确认失败** — `NODE_PATH=$(npm root -g) vitest run --root web`，预期 FAIL（文件不存在）。
- [ ] **Step 3: 最小实现** — 4 组件 + api.ts + app.ts；index.html 引用 `/mytun-web.js`；2s 轮询 status/logs。
- [ ] **Step 4: 运行确认通过** — vitest 全 PASS；`node scripts/build_web.cjs` 产出 bundle；`go build ./...` 通过（embed 生效）。
- [ ] **Step 5: Commit** — `git add web scripts Makefile .gitignore internal/web && git commit -m "feat: add webcomponents panel with tests and embed"`。
