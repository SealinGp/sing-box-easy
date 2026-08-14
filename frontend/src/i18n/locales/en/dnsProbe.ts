// DNS route inspector: probe results and the rule-flow diagram.
export default {
  title: 'DNS lookup',
  desc: 'Resolve a domain the way sing-box does, and see which rule handled it.',
  openDiagnostics: 'Diagnostics',
  domain: 'Domain',
  type: 'Type',
  run: 'Look up',
  running: 'Resolving…',
  hint: 'Enter a domain to see how it resolves through the current configuration.',
  compareServers: 'Also query each configured server directly, to compare their answers',
  answer: 'Answer',
  noAnswer: 'The query returned no records.',
  routing: 'Routing',
  ruleEvaluation: 'Rule evaluation',
  upstreams: 'Configured servers',
  disagreement: 'Servers disagree',
  server: 'Server',
  ruleNumber: 'Rule #{index}',
  noRuleMatched: 'No rule matched — the query falls through to dns.final.',
  noConditions: '(no conditions)',
  confirmedBySingBox: 'Confirmed by sing-box',
  inexact:
    'Predicted. {count} earlier rule(s) could not be evaluated here, and any of them may have matched first — enable debug logging for an exact answer.',
  cannotEvaluate: 'needs runtime state: {fields}',
  skip: {
    detour: 'only reachable through detour {detail}',
    unsupported_type: 'type {detail} has no comparable upstream',
  },
  log: {
    no_lines:
      'sing-box logged no rule decision for this query — either no rule matched, the answer was cached, or log.level is not debug.',
    ambiguous:
      'Other DNS activity was logged at the same moment, so the decisions below may not all belong to this query.',
    readError: 'Could not read the sing-box log: {error}',
  },
  state: {
    matched: 'matched',
    not_matched: 'no match',
    unevaluated: 'cannot evaluate',
  },
  toast: {
    failed: 'DNS lookup failed',
  },
}
