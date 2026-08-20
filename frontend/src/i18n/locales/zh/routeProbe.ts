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

  ladder: '规则链',
  decidedBy: '由规则 {index} 决定',
  noRuleMatched: '没有规则命中',
  address: '地址',
  resolvedBySingBox: '由 sing-box 解析',
  resolveFailed: '域名解析失败（{error}），依赖地址的规则无法判定。',
  continuesMatching: '命中但继续匹配',
  couldNotEvaluate: '无法判定：{fields}',
  inexact: '仅供参考：该判定之前有 {count} 条规则无法判定，其中任意一条都可能先命中。',
  ruleSetsUnavailable: '以下规则集无法读取，引用它们的规则无法判定：',

  state: {
    matched: '命中',
    not_matched: '未命中',
    unevaluated: '无法判定',
    skipped: '未到达',
  },

  outboundSource: {
    rule: '来自规则',
    'route.final': '来自 route.final',
    first_outbound: '第一个出站（未设置 route.final）',
    implicit_direct: '隐式 direct（未配置任何出站）',
  },

  ruleSetReason: {
    unknown_tag: '配置中不存在该标签的规则集',
    not_cached: '尚未下载 —— 请在「规则集」标签页更新',
    cache_unavailable: '无法读取 sing-box 缓存文件',
    cache_disabled: 'experimental.cache_file 未开启，远程规则集仅存在于内存中',
    file_missing: '本地文件不存在',
    unsupported_srs_version: '由更新版本的 sing-box 生成，本面板无法解析',
    parse_error: '内容无法解码',
  },

  toast: {
    failed: '模拟失败',
  },
}
