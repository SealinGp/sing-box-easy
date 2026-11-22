# 基于质量历史的自动选择

## 与 sing-box URLTest 的区别

已核对 sing-box v1.14.1：URLTest 通过节点发送 HTTP HEAD 请求，记录最新成功延迟，失败则删除该节点的内存测速结果。
选择时从当前节点开始遍历候选，只有新候选延迟加 tolerance 小于当前候选延迟才替换；并非历史达标率排名。
因此 tolerance 为 200ms 时，当前 300ms、另一个节点 150ms 不会触发替换。没有有效测速结果时回退到首个支持该网络的节点。
TCP 和 UDP 分别选择；启动时测试，活跃时按 interval 测试，超过 idle_timeout 后停止周期测试，后续使用再恢复。
默认 interval=3m、tolerance=50ms、idle_timeout=30m。测试并发数为 10。

- 选择与调度源码：https://github.com/SagerNet/sing-box/blob/v1.14.1/protocol/group/urltest.go
- 延迟测量源码：https://github.com/SagerNet/sing-box/blob/v1.14.1/common/urltest/urltest.go
- 配置文档：https://sing-box.sagernet.org/configuration/outbound/urltest/

## 面板内置质量自动选择

仅对 Selector 启用；URLTest 的内核选择保持独立。默认关闭，各组分别配置。
在代理分组头部点击“手动选择”，启用“按质量自动切换”，设置评估与切换间隔（1–1440 分钟，默认 5 分钟）。

每轮使用已有数据库历史，不额外测速，不改变自动探测调度：

1. 仅考虑当前组的实际代理节点；排除嵌套分组、direct、reject 及已移除节点。
2. 最近 N 个样本须齐全，达标率达到设置页水位（成功且延迟 ≤ 标准；超时计入分母）。
3. 最新一次探测须成功且延迟大于零；最新数据不得超过 `max(15 分钟, 2 × 自动探测间隔)`。
4. 当前节点仍合格就保持；否则按达标次数降序、成功探测 P95 延迟升序、节点 tag 升序选择。
5. 没有合格候选时保持原选择并显示原因；控制器或数据库失败则报错，下一周期重试。
6. 应用前重新读取运行分组，确认类型、成员及当前选择未变化。后台切换记录应用日志。

这里的间隔是评估与可能切换的间隔，不保证每轮都切换。历史达标率不是未来质量保证。
手动点击节点会先持久化关闭该组自动选择，再尝试手动切换；手动切换失败时仍保持自动选择关闭。
设置关闭不要求核心在线。关闭页面不影响后台工作；面板服务停止后不会继续自动选择。

## 存储与接口

策略存入 SQLite settings 的 `proxy_auto_selection`，不写入 sing-box config.json，也不自动重启内核。
策略在面板重启后恢复；最近评估和切换状态仅在本次面板进程中保留。
sing-box 自身是否保留所选成员仍由 cache_file 决定。

- 管理员：`PUT /api/v1/runtime/proxies/auto-selection`
- 请求：`{"group":"代理组","enabled":true,"interval_seconds":300}`
- 代理列表中每个已配置组新增 `auto_selection`，包含策略、原因、最近评估、下次评估、最近切换和错误。

未知/不足/过期数据不会生成自动兜底。订阅被删除或成员变化后，下一轮按运行分组的最新成员重新计算。
只有嵌套分组的 Selector 暂无可供本功能评估的直接候选，应在其下游 Selector 开启，或另建实际节点组成的 Selector。
