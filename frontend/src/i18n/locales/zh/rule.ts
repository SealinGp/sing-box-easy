/**
 * “当……则……”规则预览的共享词汇。
 *
 * 单独一个命名空间：路由规则与 DNS 规则对话框共用同一个 RuleFlowPreview，
 * 连接词（并且 / 是以下任一 / 兜底告警）两边完全一致，只有“结果”那句话由各自的
 * 命名空间解析。
 */
export default {
  flow: {
    and: '并且',
    anyOf: '是以下任一',
    more: '等 {count} 项',
    when: '当',
    everything: '任意请求 —— 此规则没有任何条件',
    invertPrefix: '非',
    catchAll: '此规则匹配所有请求并在此终止，它下面的规则永远不会执行。请添加条件，或把它移到列表末尾。',
    continues: '（继续匹配下一条规则）',
    then: { label: '则' },
  },
}
