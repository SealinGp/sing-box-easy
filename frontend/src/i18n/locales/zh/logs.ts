// 实时日志查看页（Logs.vue）。
export default {
  title: '实时日志',
  live: '实时',
  paused: '已暂停',
  disconnected: '已断开',
  pause: '暂停',
  resume: '继续',
  clear: '清空',
  jumpToBottom: '跳到底部',
  lineCount: '{n} / {max} 行',
  empty: '暂无日志。',
  sourceNone: '查看实时日志需要 sing-box 以 systemd（journald）方式运行，或已配置 log.output 日志文件。',
  sourceFile: '正在读取已配置的 log.output 日志文件。',
  sourceSyslog: '正在从系统日志（logread）读取 sing-box 日志。',
  startupFailure: {
    title: 'sing-box 启动失败',
    hint:
      '该行已被固定显示：在 debug 级别下它会在几秒内被刷走。远程规则集下载失败会直接导致启动中断 —— 请在「路由 → 规则集」中检查其下载出站。',
    dismiss: '关闭',
  },
  toast: {
    fetchFailed: '获取日志失败',
  },
}
