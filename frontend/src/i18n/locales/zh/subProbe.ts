// 订阅质量指标 —— 随时间采样的可用率与延迟。
export default {
  column: '质量',
  never: '尚未探测',
  disabled: '已关闭探测',
  nodesTested: '{reachable}/{total} 个节点',
  openDetail: '查看 {name} 的质量历史',

  range: {
    '1h': '1时',
    '6h': '6时',
    '24h': '24时',
    '7d': '7天',
    '30d': '30天',
  },

  chart: {
    availability: '可用率',
    latency: '平均延迟',
    availabilityAria: '可用率随时间变化，即可连通节点的百分比',
    latencyAria: '平均延迟随时间变化，单位毫秒',
    empty: '暂无测量数据',
    emptyHint: '探测器按设定间隔采样，首次运行在启动后不久进行。',
    hoverHint: '将鼠标移到图表上查看某一时刻',
    noLatency: '无节点响应',
  },

  dialog: {
    title: '质量 —— {name}',
    noData: '暂无测量数据',
    probeNow: '立即探测',
    target: '测试目标',
    nodes: '最近一次探测的各节点结果',
    nodesUnavailable: '面板启动后尚无探测记录。可立即探测，或等待下一轮。',
    untestable: '无法测试',
    unreachable: '不可达',
  },

  form: {
    enabled: '测量质量',
    enabledHint: '定期对该订阅的节点做 URL 测试，记录可用率与延迟。',
    url: '延迟测试地址',
    urlHint:
      '必须是 https —— sing-box 会忽略 http:// 地址并静默改用自带默认值。留空则使用 https://www.gstatic.com/generate_204。',
  },

  settings: {
    title: '节点质量探测',
    desc: '订阅节点多久做一次 URL 测试，以及保留多少历史数据。',
    interval: '探测间隔',
    intervalHint: '取值范围 {min} 到 {max}。',
    timeout: '单节点超时',
    timeoutHint: '单位毫秒。受 Clash API 请求超时限制，上限为 {max}。',
    retention: '历史保留时长',
    retentionDays: '{days} 天',
    maxPoints: '每个订阅最多保留',
    maxPointsHint: '两个上限中先达到的那个生效。',
    stored: '当前已存储',
    storedValue: '{count} 条采样（约 {size}）',
    requiresClashApi:
      '探测使用 sing-box 自带的 URL 测试，因此需要设置 experimental.clash_api.external_controller 并保持 sing-box 运行。',
    save: '保存',
  },

  validation: {
    urlScheme: '必须是 https:// 地址 —— sing-box 会忽略 http:// 目标并改用自带默认值。',
    urlInvalid: '请输入有效的地址',
  },

  notify: {
    historyFailed: '加载质量历史失败',
    nodesFailed: '加载节点结果失败',
    probeFailed: '探测失败',
    probed: '探测完成：{reachable}/{total} 可达，平均 {latency}',
    settingsSaved: '探测设置已保存',
    settingsFailed: '保存探测设置失败',
  },
}
