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

每轮主动探测当前节点，单次超时 5 秒，失败立即复测。两次均失败时以最多 5 个并发探测本组实际节点，并选择本轮成功且延迟最低的候选。故障恢复允许使用历史不足或过期的节点，但必须有本轮成功结果。单组评估总预算 60 秒。控制器错误会报告失败。

正常节点的质量评估继续使用最近 N 条历史。当前节点达到推荐水位则保持；否则按达标率、成功探测 P95 选择候选。全组未达水位时，最高达标率 1 个百分点以内视为同档，按 P95 选择；全组零达标时先比较成功率。

质量优化需要连续两轮选中同一候选，且距上次切换至少 5 分钟。切换前实测候选，必须比本轮当前节点至少快 20% 且快 50 ms。故障恢复直接执行，无需等待这两个条件。

探测结果存入节点历史，来源显示为“自动切换探测”。网络或控制器未返回确定的测速结果时不会写入失败样本。切换前重新读取分组，确认当前选择及目标成员。手动选择会关闭本组自动切换。

页面把当前节点排在最前，并显示最近一次自动切换的起点、终点和时间。记录保存在面板进程内存中，重启后重新积累。

## 存储与接口

策略存入 SQLite settings 的 `proxy_auto_selection`，不写入 sing-box config.json，也不自动重启内核。
策略在面板重启后恢复；最近评估和切换状态仅在本次面板进程中保留。
sing-box 自身是否保留所选成员仍由 cache_file 决定。

- 管理员：`PUT /api/v1/runtime/proxies/auto-selection`
- 请求：`{"group":"代理组","enabled":true,"interval_seconds":300}`
- 代理列表中每个已配置组新增 `auto_selection`，包含策略、原因、最近评估、下次评估、最近切换和错误。

订阅被删除或成员变化后，下一轮按运行分组的最新成员重新计算。
只有嵌套分组的 Selector 暂无可供本功能评估的直接候选，应在其下游 Selector 开启，或另建实际节点组成的 Selector。
