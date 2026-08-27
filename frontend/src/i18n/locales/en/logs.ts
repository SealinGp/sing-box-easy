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
  startupFailure: {
    title: 'sing-box failed to start',
    hint:
      'This line is pinned because at debug level it scrolls out of view within seconds. A remote rule set that cannot be downloaded aborts startup \u2014 check its download detour under Route \u2192 Rule Sets.',
    dismiss: 'Dismiss',
  },
  toast: {
    fetchFailed: 'Failed to fetch logs',
  },
  transport: {
    // Shown only when the stream could not be used.
    poll: 'polling (stream unavailable)',
  },
  tabs: {
    singbox: 'sing-box',
    app: 'sing-box-easy',
  },
  // The panel's own log lives in an in-process ring buffer, so its limitation
  // is permanent rather than a misconfiguration — hence a note that is always
  // shown, not one that appears when something is wrong.
  sourceMemory: 'The panel\u2019s own log is kept in memory and starts empty after a restart or an update.',
  emptyApp: 'No panel log lines yet.',
}
