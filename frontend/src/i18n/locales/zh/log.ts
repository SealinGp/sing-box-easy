// 日志配置页（Log.vue）。
export default {
  title: '日志配置',
  subtitle: '配置 sing-box 日志设置',
  loading: '正在加载日志配置...',
  disabled: {
    label: '禁用日志',
    help: '关闭所有日志输出',
  },
  level: {
    label: '日志级别',
    help: '设置要显示的最低日志级别',
  },
  output: {
    label: '输出文件',
    help: '日志文件路径（留空则输出到 stdout）',
    placeholder: '/var/log/sing-box.log',
  },
  timestamp: {
    label: '包含时间戳',
    help: '为日志条目添加时间戳',
  },
  resetDefaults: '恢复默认',
  saveConfig: '保存配置',
  toast: {
    loadFailed: '加载日志配置失败',
    savedOk: '日志配置保存成功',
    saveFailed: '保存日志配置失败',
  },
}
