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
  ruleSetsUnavailable: 'Some rule sets could not be read, so rules using them could not be decided:',


  outboundSource: {
    rule: 'from a rule',
    'route.final': 'from route.final',
    first_outbound: 'first outbound (no route.final set)',
    implicit_direct: 'implicit direct (no outbounds configured)',
  },

  ruleSetReason: {
    unknown_tag: 'no rule set with this tag exists in the config',
    not_cached: 'never downloaded — update it on the Rule Sets tab',
    cache_unavailable: 'the sing-box cache file could not be read',
    cache_disabled: 'experimental.cache_file is off, so remote sets are only in memory',
    file_missing: 'the local file is missing',
    unsupported_srs_version: 'built by a newer sing-box than this panel understands',
    parse_error: 'the content could not be decoded',
  },

  toast: {
    failed: 'Simulation failed',
  },
}
