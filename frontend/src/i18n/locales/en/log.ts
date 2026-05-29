// Log configuration page (Log.vue).
export default {
  title: 'Log Configuration',
  subtitle: 'Configure sing-box logging settings',
  loading: 'Loading log configuration...',
  disabled: {
    label: 'Disable Logging',
    help: 'Turn off all logging output',
  },
  level: {
    label: 'Log Level',
    help: 'Set the minimum log level to display',
  },
  output: {
    label: 'Output File',
    help: 'Path to log file (leave empty for stdout)',
    placeholder: '/var/log/sing-box.log',
  },
  timestamp: {
    label: 'Include Timestamp',
    help: 'Add timestamps to log entries',
  },
  resetDefaults: 'Reset to Defaults',
  saveConfig: 'Save Configuration',
  toast: {
    loadFailed: 'Failed to load log configuration',
    savedOk: 'Log configuration saved successfully',
    saveFailed: 'Failed to save log configuration',
  },
}
