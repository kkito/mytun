# AGENTS.md — mytun 项目规则

## 1. TDD 是强制流程（无例外）

所有功能开发和 bug 修复都必须走 TDD（见 skill `test-driven-development`）：

- 先写失败的测试（RED），**亲眼看到它按预期原因失败**，再写最少量的生产代码（GREEN），最后重构（REFACTOR）。
- 禁止事后补测试、禁止"先实现再补单测"——测试一次通过证明不了任何事。
- bug 修复必须先有复现该 bug 的失败测试，修完后该测试转为回归测试。
- 收尾检查：`make test-all`（= `go test ./...` + 前端 `vitest run`）必须全绿才可提交。

## 2. 无状态函数优先

- 核心逻辑一律写成**无状态纯函数**（输入 → 输出，不碰全局/静态可变状态）：包解析、校验和、格式化、信封组装都属此类。
- 必须带状态的部分（TUN 句柄、计数器、日志环、锁）压成**最薄的几行胶水**：只做加锁、查表、委托，判定逻辑下沉到纯函数。
- 新增全局可变状态需在提交说明里解释为什么不能用参数传递；测试必须能用"构造输入 → 断言输出"覆盖，不依赖进程残留状态。
- Gin handler 只做三件事：绑参与校验、调 `internal/*` 纯逻辑/管理器、写统一信封；禁止在 handler 里写业务判定。

## 3. 做完必须做重构审视

功能/bug 完成后，提交前必须过一遍：

1. 是否有重复逻辑（相同 `if` 模式、相同字符串拼接、相同错误处理）可以合并为一个函数？
2. 该函数是否放错了层——是否该下沉到所属包（如包构造归 `internal/echo` 或新建 `internal/packet`，而非散在 web handler 调用点）？
3. 跨包复用的逻辑是否该抽到公共位置（`internal/` 下的新包），而不是 copy-paste？
4. 测试钩子是否暴露了生产代码的状态管理缺陷——能顺手修就顺手修。

YAGNI 仍然有效：只抽象**已经出现两次**的重复，不为假想的未来做设计。

## 4. 单测零网络、零 root（无例外）

单测禁止发起真实网络调用、禁止建真实 TUN（需 root/`/dev/net/tun`/`utun`）：有必要直接 mock/fake。

- 平台边界必须可注入：`internal/tun` 的真实建卡只在 e2e 脚本里跑，单测用 fake Device 或接口抽象。
- API 单测用 `gin.CreateTestContext + httptest`，不绑定真实端口、不起真实 TUN。
- 用"哨兵 fake"（被调用即 `t.Fatal`）锁住零 root：纯逻辑用例注入哨兵，漏调建卡立刻变红。
- 需要 root 的验证走 `scripts/darwin_e2e.sh`（macOS 实机），不进 `go test`。

## 5. 前端开发套路（从 rybaproxy 复制）

- 默认本机安装了 node 环境；全局工具链（**不用安装在当前项目里，不写 `web/package.json`、不提交 `web/node_modules`**）：
  ```sh
  npm install -g vitest happy-dom esbuild typescript
  ```
  - `vitest`：单元测试 runner（`NODE_PATH=$(npm root -g) vitest run`）
  - `happy-dom`：vitest 的 DOM 环境（`vitest.config.mjs` 里 `environment: 'happy-dom'`，跑 WebComponent 测试用）
  - `esbuild`：TS 打包（`scripts/build_web.cjs` 把 `web/src/app.ts` 打成 `build/web-dist/mytun-web.js`，IIFE 全局 `MytunWeb`），Go 端 `go:embed` 打进二进制
  - `typescript`：类型检查（`tsc --noEmit -p web/tsconfig.json`，可选，不参与打包）
- 页面通过 WebComponent 实现，一个组件一文件（如 `web/src/tun-status.ts`）。
- 通过 vitest 和 happy-dom 做完整的页面组件单元测试（`web/tests/*.test.ts`），先 RED 后 GREEN。
- 前后端分离，通过 fetch API 调用接口；API 信封与 rybaproxy 对齐：`{ok:true,data}` / `{ok:false,error:{code,message}}`。
- 和当前项目组装：`scripts/build_web.cjs` 产物目录为 `build/web-dist`（不污染源码树，不提交；`dist/`、`build/` 都在 `.gitignore`）。
