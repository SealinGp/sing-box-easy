// The route topology card (RouteTopologyCard.vue / RouteFlowDiagram.vue).
export default {
  title: 'Traffic flow',
  desc: 'Every inbound enters the same rule ladder. sing-box stops at the first rule that matches, so the order is the logic.',
  ariaLabel: 'Diagram of inbounds, routing rules and outbounds',
  openRoute: 'Edit rules',
  refresh: 'Reload from config',
  loadFailed: 'Could not read the config',
  empty: 'No routing rules configured — everything falls through to the default outbound.',
  noConfig: 'No config to draw yet.',

  // Column strip under the diagram, naming what the reader is looking at.
  columns: {
    inbounds: 'Inbounds',
    rules: 'Rules — top down, first match wins',
    outbounds: 'Outbounds',
  },

  // Rule row + tooltip wording.
  when: 'when',
  then: 'then',
  and: 'and',
  not: 'NOT',
  anyTraffic: 'any traffic',
  scopedTo: 'Only from: {inbounds}',
  catchAll: 'No conditions — this matches everything below it.',
  everythingElse: 'Everything else',

  // Exit node second line.
  members: '{n} member | {n} members',
  reachedBy: 'Reached by rule {rules} | Reached by {n} rules: {rules}',
  exitKind: {
    reject: 'reject · {method}',
    hijackDns: 'answered by DNS',
    missing: 'no such outbound',
    implicit: 'synthesised by sing-box',
  },

  // The three cases sing-box collapses into one word.
  fallthroughSource: {
    final: 'route.final',
    firstOutbound: 'No route.final — the first outbound is the default',
    implicitDirect: 'No outbounds — sing-box synthesises a direct one',
  },

  legend: {
    converge: '{n} rules, one outbound',
    passthrough: 'Does not stop matching — the rules below still run',
    unreachable: 'Never reached: an earlier rule matches everything',
    missing: 'Names an outbound that does not exist',
    final: 'Fall-through',
    scoped: 'Restricted to certain inbounds',
  },

  // Warnings surfaced above the diagram, where they are actionable.
  warnUnreachable: '{n} rule is never reached | {n} rules are never reached',
  warnMissing: '{n} rule points at an outbound that does not exist | {n} rules point at outbounds that do not exist',

  // The live overlay: real connections lighting the expected drawing.
  live: {
    toggle: 'Live',
    on: 'Watching live traffic — click to stop',
    off: 'Light the diagram with real traffic',
    needsRunning: 'Start sing-box to watch live traffic',
    connecting: 'Connecting…',
    failed: 'Live traffic unavailable',
    retrying: 'Retrying…',
    down: 'Down',
    up: 'Up',
    connections: '{n} connection | {n} connections',
    shownOf: '{shown} of {all}',
    closed: '{n} closed',
    filterSource: 'Source IP',
    allSources: 'All devices',
    // The picker lists the clients holding connections right now, so the count
    // is what tells one busy device from an idle one.
    sourceConnections: '{n} conn',
    noSources: 'No devices yet',
    filterHost: 'Host contains…',
    clearFilter: 'Clear',
    unmatched: '{n} connection lit by outbound only — its rule is not in the running list (sing-box was reloaded) | {n} connections lit by outbound only — their rules are not in the running list (sing-box was reloaded)',
    via: 'via {tag}',
    legendMoving: 'Pulsing: top {n} by download',
    legendLit: 'Lit: carrying traffic',
    idle: 'No traffic in the last second',
    rate: '{rate}/s',
    hosts: 'Top destinations',
  },

  // In-window full-size mode for the diagram.
  fullWindow: {
    enter: 'Show full window',
    exit: 'Back to the card',
    restore: 'Restore',
    escHint: 'Esc to close',
  },
}
