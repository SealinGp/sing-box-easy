// DNS 规则流程图。
export default {
  title: 'DNS 规则流程',
  desc: 'sing-box 自上而下依次匹配这些规则，命中第一条后即停止。',
  empty: '尚未配置 DNS 规则，所有查询将直接交给 dns.final。',
  query: '查询',
  entersRules: '进入规则列表',
  final: 'dns.final',
  servers: '服务器',
  matched: '已命中',
  predicted: '推测命中',
  runtimeOnly: '需要运行时状态：{fields}',
  loadFailed: '加载 DNS 配置失败',
  noConditions: '（无条件）',
}
