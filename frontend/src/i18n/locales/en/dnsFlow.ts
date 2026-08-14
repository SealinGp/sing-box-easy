// The DNS rule ladder diagram.
export default {
  title: 'DNS rule flow',
  desc: 'sing-box checks these rules top to bottom and stops at the first match.',
  empty: 'No DNS rules are configured, so every query goes straight to dns.final.',
  query: 'Query',
  entersRules: 'enters the rule list',
  final: 'dns.final',
  servers: 'Servers',
  matched: 'matched',
  predicted: 'predicted',
  runtimeOnly: 'needs runtime state: {fields}',
  loadFailed: 'Failed to load the DNS configuration',
}
