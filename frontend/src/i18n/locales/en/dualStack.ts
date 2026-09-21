export default {
  title: 'IPv4 / IPv6 domain test',
  desc: 'Resolve a domain the way sing-box does, then send real traffic to each address family and report which rule carried it.',

  domains: 'Domains',
  domainsHint: 'One per line, or comma separated. At most {max}.',
  placeholder: 'www.baidu.com\nwww.google.com',
  run: 'Run test',
  running: 'Testing…',
  hint: 'Enter one or more domains to test both address families.',

  showAdvanced: 'Show options',
  hideAdvanced: 'Hide options',
  fields: {
    port: 'Port',
    tls: 'Complete a TLS handshake',
    skipDial: 'Do not send traffic',
    timeout: 'Per-dial timeout (ms)',
  },
  tlsHelp: 'Proves the path reaches the intended server, not just something listening.',
  skipDialHelp: 'Report DNS answers and routing predictions only.',

  stage: {
    endpoint: 'Reading the config…',
    resolve: 'Resolving through sing-box…',
    predict: 'Walking the route rules…',
    traffic: 'Sending traffic…',
  },

  endpoint: {
    title: 'Client DNS',
    inbound: 'Hijacked into inbound {inbound} at {address}',
    protocol_dns: 'Hijacked on every inbound by a protocol=dns rule — no single address to point clients at',
    none: 'No route rule hijacks DNS, so clients resolve without sing-box',
    unresolved: 'Inbound {inbound} has no listen port, so there is no address to query',
  },

  client: {
    noIpv6: 'This host has no global IPv6 address, so the IPv6 rows were not tested.',
    noIpv4: 'This host has no global IPv4 address, so the IPv4 rows were not tested.',
  },

  family: {
    ipv4: 'IPv4',
    ipv6: 'IPv6',
  },

  status: {
    reachable: 'reachable',
    unreachable: 'unreachable',
    no_address: 'no address',
    untested: 'not tested',
  },
  statusHelp: {
    no_address: 'sing-box returned no record of this type. For a proxied domain this is what an IPv6 split is supposed to do.',
    untested: 'Nothing was dialled, so this says nothing about whether the path works.',
  },

  agreement: {
    agree: 'as predicted',
    differ: 'NOT as predicted',
    unknown: 'unverified',
  },
  differHelp: 'The config predicted {predicted} but the traffic left through {observed}.',
  inexact: 'The prediction was inexact — an undecidable rule sat ahead of the decision — so a mismatch here may be the prediction’s fault rather than the config’s.',

  predicted: 'predicted',
  observed: 'observed',
  via: 'via',
  noObservation: 'no matching connection appeared in sing-box',
  observeUnavailable: 'Connections could not be read, so nothing could be confirmed: {error}',

  toast: {
    failed: 'Test failed',
  },
}
