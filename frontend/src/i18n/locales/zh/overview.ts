// 仪表盘概览页（Overview.vue）。
export default {
  title: '仪表盘概览',
  serviceStatus: '服务状态',
  pid: 'PID',
  uptime: '运行时长',
  refreshStatus: '刷新状态',
  status: {
    running: '运行中',
    stopped: '已停止',
    unknown: '未知',
  },
  toast: {
    startedOk: '服务启动成功',
    stoppedOk: '服务停止成功',
    restartedOk: '服务重启成功',
    fetchFailed: '获取服务状态失败',
    startFailed: '启动服务失败',
    stopFailed: '停止服务失败',
    restartFailed: '重启服务失败',
  },
}
