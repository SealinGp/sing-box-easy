// Live log viewer page (Logs.vue).
export default {
  title: 'Live Logs',
  live: 'Live',
  paused: 'Paused',
  disconnected: 'Disconnected',
  pause: 'Pause',
  resume: 'Resume',
  clear: 'Clear',
  jumpToBottom: 'Jump to bottom',
  lineCount: '{n} / {max} lines',
  empty: 'No log lines yet.',
  sourceNone: 'Live logs require sing-box to run under systemd (journald) or to have a log.output file configured.',
  sourceFile: 'Reading from the configured log.output file.',
  sourceSyslog: 'Reading sing-box entries from the system log (logread).',
  toast: {
    fetchFailed: 'Failed to fetch logs',
  },
}
