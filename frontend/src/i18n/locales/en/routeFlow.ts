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
}
