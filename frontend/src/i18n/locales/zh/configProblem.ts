// 配置预检查对话框（ConfigProblemDialog.vue）：某项设置指向了不存在或为空的出站。
export default {
  title: '配置存在问题',
  intro: '下列设置指向了不可用的出站。sing-box 的 check 命令检查不出这类问题，但 sing-box 启动时会因此失败，所以面板已拦截本次操作。',
  unchanged: '当前生效的配置没有被修改。',
  fields: {
    'route.final': '默认出站（route.final）',
    'route.rules': '路由规则',
    'route.rule_set': '规则集下载出站',
    'dns.servers': 'DNS 服务器出站',
  },
  fix: {
    'route.final': '去选择默认出站',
    'route.rules': '去修改路由规则',
    'route.rule_set': '去修改规则集',
    'dns.servers': '去修改 DNS 服务器',
  },
  kinds: {
    missing_outbound: '出站「{tag}」不存在',
    empty_group: '出站组「{tag}」中没有任何节点',
  },
}
