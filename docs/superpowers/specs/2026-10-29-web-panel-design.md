# mytun Web 面板 Design

日期：2026-10-29 · 状态：approved by user（单 TUN + Gin + WebComponent，对齐 rybaproxy）

## 目标

给 mytun 加一个 Web 操作和展示面板：创建/删除 TUN、发送包并看到接收数据。
API 用 Gin，前端用 WebComponent（复刻 rybaproxy 套路），前后端分离、各自可单测。

## 架构

浏览器（4 个 WebComponent，2s fetch 轮询）→ Gin `:8080`（`/api/*` + `/healthz`）
→ 新增 `internal/tunmgr`（mutex 管理单 TUN 实例：down/creating/up）
→ 复用现有 `internal/tun`（建卡/读写）+ `internal/netconf`（配地址）+ `internal/echo`（回包）。

```
web/src/*.ts --fetch--> Gin /api/* --> tunmgr --> tun/netconf/echo
                              |
                         stats + 环形日志（200 条）
```

CLI 保持可用：`mytun --web :8080` 同时跑 TUN+Web；`--web ""` 纯 CLI 老行为。

## API（5 个，信封对齐 rybaproxy）

统一信封 `{ok:true,data}` / `{ok:false,error:{code,message}}`；
bad json→400、无 tun→404、重复创建→409。

- `GET /healthz` → `{status:"ok", version}`（无鉴权，对齐 ryba）
- `GET /api/status` → `{state, ifname, addr, mtu, rx_packets, tx_packets, rx_bytes, tx_bytes}`
- `POST /api/tun` `{addr, mtu}` → 创建并 up；已存在 409 `TUN_EXISTS`
- `DELETE /api/tun` → 删除并 down；不存在 404 `NO_TUN`
- `POST /api/send` `{dst, size?, proto?}` → 从 TUN 口构造 IPv4 包写入；`GET /api/logs?limit=` 看收到的包

收包循环不动，加一行"记 stats + 推环形 log"；echo 开关复用现有逻辑。

## 前端（4 组件，对标 ryba ryba-card/stat）

- `web/src/api.ts` — fetch 封装 + 信封解析（直接抄 ryba 的 `api.ts`）
- `<tun-status>` — 状态卡（接口名/地址/MTU/收发计数）
- `<tun-controls>` — 创建/删除表单 + echo 开关
- `<tun-send>` — 发送表单（目的 IP/大小）+ 结果
- `<tun-log>` — 收包日志滚动（轮询 `/api/logs`）
- `web/src/app.ts` — 入口，esbuild 打成 `build/web-dist/mytun-web.js`（IIFE `MytunWeb`），Go `go:embed`

工具链：全局 `vitest happy-dom esbuild typescript`（不写 web/package.json，不提交 node_modules），
详见 AGENTS.md §5。

## 测试

- Go：`gin.CreateTestContext + httptest` 覆盖 5 个 handler（含 409/404/400）；tunmgr 用 fake device 测状态机（零 root）。
- 前端：vitest + happy-dom，每个组件一个 `web/tests/*.test.ts`。
- `make test-all` = `go test ./...` + `vitest run`；需 root 的走 `scripts/darwin_e2e.sh`，不进单测。

## 文件清单（实现时创建）

- `internal/tunmgr/manager.go`（状态机 + stats + 环形 log）+ 测试
- `internal/web/server.go`（Gin 路由 + go:embed）+ 5 个 `api_*.go` handler + `*_test.go`
- `web/src/{api,app,tun-status,tun-controls,tun-send,tun-log}.ts`
- `web/{index.html,style.css,tsconfig.json,vitest.config.mjs}`
- `web/tests/{api,tun-status,tun-controls,tun-send,tun-log}.test.ts`
- `scripts/build_web.cjs` + `Makefile` 新增 `web-build/test-all` 目标
- `go.mod` 新增 `github.com/gin-gonic/gin`
