// Subscription quality metrics — availability + latency sampled over time.
export default {
  // Column + summary shown on the Subscriptions page and the Overview card.
  column: 'Quality',
  never: 'Not probed yet',
  disabled: 'Probing off',
  nodesTested: '{reachable}/{total} nodes',
  openDetail: 'Quality history for {name}',

  range: {
    '1h': '1h',
    '6h': '6h',
    '24h': '24h',
    '7d': '7d',
    '30d': '30d',
  },

  chart: {
    availability: 'Availability',
    latency: 'Average latency',
    availabilityAria: 'Availability over time, as a percentage of nodes reachable',
    latencyAria: 'Average latency over time, in milliseconds',
    empty: 'No measurements yet',
    emptyHint: 'The prober samples on its interval; the first run happens shortly after startup.',
    hoverHint: 'Hover the chart to read a moment',
    noLatency: 'no nodes answered',
  },

  dialog: {
    title: 'Quality — {name}',
    noData: 'No measurements yet',
    probeNow: 'Probe now',
    target: 'Tested against',
    nodes: 'Latest run, per node',
    nodesUnavailable: 'No run recorded since the panel started. Probe now, or wait for the next sweep.',
    untestable: 'Not testable',
    unreachable: 'Unreachable',
  },

  form: {
    enabled: 'Measure quality',
    enabledHint:
      'Periodically URL-tests this subscription\'s nodes to track availability and latency.',
    url: 'Latency test URL',
    urlHint:
      'Must be https — sing-box ignores http:// targets and silently tests its own default instead. Leave empty for https://www.gstatic.com/generate_204.',
  },

  settings: {
    title: 'Node quality probing',
    desc: 'How often subscription nodes are URL-tested, and how much history is kept.',
    interval: 'Probe interval',
    intervalHint: 'Between {min} and {max}.',
    timeout: 'Per-node timeout',
    timeoutHint: 'Milliseconds. Capped at {max} by the Clash API request timeout.',
    retention: 'Keep history for',
    retentionDays: '{days} days',
    maxPoints: 'Max samples per subscription',
    maxPointsHint: 'Whichever bound is reached first applies.',
    stored: 'Currently stored',
    storedValue: '{count} samples (~{size})',
    requiresClashApi:
      'Probing uses sing-box\'s own URL test, so it needs experimental.clash_api.external_controller set and sing-box running.',
    save: 'Save',
  },

  validation: {
    urlScheme: 'Must be an https:// URL — sing-box ignores http:// targets and tests its own default instead.',
    urlInvalid: 'Enter a valid URL',
  },

  notify: {
    historyFailed: 'Failed to load quality history',
    nodesFailed: 'Failed to load node results',
    probeFailed: 'Probe failed',
    probed: 'Probed: {reachable}/{total} reachable, {latency} average',
    settingsSaved: 'Probe settings saved',
    settingsFailed: 'Failed to save probe settings',
  },
}
