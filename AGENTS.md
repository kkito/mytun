# mytun 规则

- 先让它跑起来：最小改动解决问题；一个决定只住一处。
- TDD：先写失败测试，亲眼看到预期失败，再写最小实现；修 bug 先复现。
- 数据形状先行：先定结构再写逻辑；让非法状态写不出来；外部输入在入口校验一次。
- 无状态优先：纯函数做判定，状态压成薄胶水；handler 只做绑定/委托/信封。
- 做减法：加东西前先删复杂度；一次只做一个可验证的小单元。
- 修根因：掩盖崩溃的 guard 不是修；测试行为不测实现。
- 单测零网络零 root；要 root 的走 scripts/*_e2e.sh；提交前 `make test-all` 全绿。
- 前端：WebComponent 一组件一文件，vitest 先 RED 后 GREEN；用全局工具链，不写 web/package.json、不提交 node_modules。
