export default {
  title: 'Route Simulator',
  desc: 'Where a destination would go, before you send anything to it. Predicted from the saved config — the Clash dashboard shows connections after they happen; this shows them before.',
  placeholder: 'example.com or 1.2.3.4',
  run: 'Simulate',
  showAdvanced: 'More options',
  hideAdvanced: 'Fewer options',
  unknownProtocol: 'Unknown',
  anyInbound: 'Any inbound',

  fields: {
    port: 'Port',
    network: 'Network',
    inbound: 'Inbound',
    protocol: 'Protocol',
    sourceIp: 'Source IP',
  },

  openSimulator: 'Full simulator',
  decidedBy: 'Decided by rule {index}',
  noRuleMatched: 'No rule matched',
  address: 'Address',
  resolvedBySingBox: 'resolved by sing-box',
  resolveFailed: 'Could not resolve the name ({error}). Address-based rules could not be decided.',
  inexact: 'Best guess: {count} rule(s) ahead of this decision could not be evaluated, and any of them could have matched first.',


  outboundSource: {
    rule: 'from a rule',
    'route.final': 'from route.final',
    first_outbound: 'first outbound (no route.final set)',
    implicit_direct: 'implicit direct (no outbounds configured)',
  },


  toast: {
    failed: 'Simulation failed',
  },
}
