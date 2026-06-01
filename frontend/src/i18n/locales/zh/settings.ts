// 设置页（Settings.vue）。
export default {
  title: '设置',
  versionHistory: {
    title: '配置版本历史',
    desc: '保留多少个历史配置。超出该数量的旧版本会在每次保存后自动清理。范围 {min}–{max}。',
    versionsToKeep: '保留版本数',
  },
  language: {
    title: '语言',
    desc: '选择界面语言。',
  },
  about: {
    title: '关于',
  },
  toast: {
    loadFailed: '加载设置失败',
    saveFailed: '保存设置失败',
    savedTitle: '已保存',
    savedDetail: '设置已更新',
  },
}
