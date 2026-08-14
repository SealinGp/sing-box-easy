// DNS 路由检查器：查询结果与规则流程图。
export default {
  title: 'DNS 查询',
  desc: '按 sing-box 的实际逻辑解析域名，并显示命中的规则。',
  openDiagnostics: '诊断',
  domain: '域名',
  type: '类型',
  run: '查询',
  running: '解析中…',
  hint: '输入域名，查看它在当前配置下如何解析。',
  compareServers: '同时直接查询各已配置的服务器，以对比其结果',
  answer: '解析结果',
  noAnswer: '该查询没有返回任何记录。',
  routing: '路由',
  ruleEvaluation: '规则匹配过程',
  upstreams: '已配置的服务器',
  disagreement: '各服务器结果不一致',
  server: '服务器',
  ruleNumber: '规则 #{index}',
  noRuleMatched: '没有规则命中——查询将落到 dns.final。',
  noConditions: '（无条件）',
  confirmedBySingBox: '已由 sing-box 确认',
  inexact:
    '此为推测结果。有 {count} 条更靠前的规则无法在此评估，其中任意一条都可能先命中——开启 debug 日志可获得确切结果。',
  cannotEvaluate: '需要运行时状态：{fields}',
  log: {
    no_lines:
      'sing-box 未记录此次查询的规则判定——可能没有规则命中、结果来自缓存，或 log.level 不是 debug。',
    ambiguous: '同一时刻还有其他 DNS 活动被记录，下方的判定未必都属于本次查询。',
    readError: '无法读取 sing-box 日志：{error}',
  },
  state: {
    matched: '命中',
    not_matched: '未命中',
    unevaluated: '无法判断',
  },
  toast: {
    failed: 'DNS 查询失败',
  },
}
