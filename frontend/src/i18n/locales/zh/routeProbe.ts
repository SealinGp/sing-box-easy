export default {
  title: '路由模拟',
  desc: '在发出流量之前，预测目标会走哪个出站。基于已保存的配置推演 —— Clash 面板显示连接发生之后的结果，这里显示之前的预测。',
  placeholder: 'example.com 或 1.2.3.4',
  run: '模拟',
  showAdvanced: '更多选项',
  hideAdvanced: '收起选项',
  unknownProtocol: '未知',
  anyInbound: '任意入站',

  fields: {
    port: '端口',
    network: '网络',
    inbound: '入站',
    protocol: '协议',
    sourceIp: '来源 IP',
  },

  openSimulator: '完整模拟',
  decidedBy: '由规则 {index} 决定',
  noRuleMatched: '没有规则命中',
  address: '地址',
  resolvedBySingBox: '由 sing-box 解析',
  resolveFailed: '域名解析失败（{error}），依赖地址的规则无法判定。',
  inexact: '仅供参考：该判定之前有 {count} 条规则无法判定，其中任意一条都可能先命中。',


  outboundSource: {
    rule: '来自规则',
    'route.final': '来自 route.final',
    first_outbound: '第一个出站（未设置 route.final）',
    implicit_direct: '隐式 direct（未配置任何出站）',
  },


  toast: {
    failed: '模拟失败',
  },
}
