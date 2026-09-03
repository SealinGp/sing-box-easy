// 流量走向卡片（RouteTopologyCard.vue / RouteFlowDiagram.vue）。
export default {
  title: '流量走向',
  desc: '所有入站都进入同一条规则链。sing-box 在第一条命中的规则处停止，因此顺序本身就是逻辑。',
  ariaLabel: '入站、路由规则与出站的关系图',
  openRoute: '编辑规则',
  refresh: '重新读取配置',
  loadFailed: '无法读取配置',
  empty: '未配置路由规则 —— 所有流量都落到默认出站。',
  noConfig: '暂无可绘制的配置。',

  columns: {
    inbounds: '入站',
    rules: '规则 —— 自上而下，首个命中生效',
    outbounds: '出站',
  },

  when: '当',
  then: '则',
  and: '且',
  not: '非',
  anyTraffic: '任意流量',
  scopedTo: '仅限入站：{inbounds}',
  catchAll: '没有任何条件 —— 它会匹配其下的所有流量。',
  everythingElse: '其余全部',

  members: '{n} 个节点',
  reachedBy: '由规则 {rules} 命中 | 由 {n} 条规则命中：{rules}',
  exitKind: {
    reject: '拒绝 · {method}',
    hijackDns: '由 DNS 应答',
    missing: '该出站不存在',
    implicit: 'sing-box 自动生成',
  },

  fallthroughSource: {
    final: 'route.final',
    firstOutbound: '未设置 route.final —— 使用第一个出站作为默认',
    implicitDirect: '没有任何出站 —— sing-box 自动生成一个 direct',
  },

  legend: {
    converge: '{n} 条规则，同一个出站',
    passthrough: '不终止匹配 —— 后面的规则继续生效',
    unreachable: '永远不会命中：前面有规则匹配所有流量',
    missing: '指向了不存在的出站',
    final: '兜底',
    scoped: '仅对部分入站生效',
  },

  warnUnreachable: '有 {n} 条规则永远不会命中',
  warnMissing: '有 {n} 条规则指向了不存在的出站',

  live: {
    toggle: '实时',
    on: '正在显示实时流量 —— 点击停止',
    off: '用真实流量点亮这张图',
    needsRunning: '启动 sing-box 后可查看实时流量',
    connecting: '连接中…',
    failed: '实时流量不可用',
    retrying: '重试中…',
    down: '下行',
    up: '上行',
    connections: '{n} 个连接',
    shownOf: '{shown} / {all}',
    closed: '{n} 个已关闭',
    filterSource: '来源 IP',
    filterHost: '目标包含…',
    clearFilter: '清除',
    unmatched: '有 {n} 个连接仅按出站点亮 —— 其规则不在当前运行的规则列表中（sing-box 已重载）',
    via: '经 {tag}',
    legendMoving: '流动：按下行速率取前 {n} 条',
    legendLit: '点亮：有流量经过',
    idle: '最近一秒没有流量',
    rate: '{rate}/s',
    hosts: '主要目标',
  },

  fullWindow: {
    enter: '全窗口显示',
    exit: '回到卡片',
    restore: '恢复',
    escHint: '按 Esc 关闭',
  },
}
