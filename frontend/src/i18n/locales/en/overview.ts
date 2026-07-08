// Dashboard Overview page (Overview.vue).
export default {
  title: 'Dashboard Overview',
  serviceStatus: 'Service Status',
  pid: 'PID',
  uptime: 'Uptime',
  lastStarted: 'Last started',
  version: 'Version',
  refreshStatus: 'Refresh Status',
  logSettings: 'Log settings',
  realtimeLog: 'Real-time log',
  status: {
    running: 'running',
    stopped: 'stopped',
    unknown: 'unknown',
  },
  toast: {
    startedOk: 'Service started successfully',
    stoppedOk: 'Service stopped successfully',
    restartedOk: 'Service restarted successfully',
    fetchFailed: 'Failed to fetch service status',
    startFailed: 'Failed to start service',
    stopFailed: 'Failed to stop service',
    restartFailed: 'Failed to restart service',
  },
}
