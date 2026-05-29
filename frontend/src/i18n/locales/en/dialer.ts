// Shared dial options block (DialerOptions.vue).
export default {
  title: 'Dial Options',
  optional: '(Optional)',
  detour: {
    label: 'Detour',
    hint: '(Prefix Outbound tag for chain proxy)',
    placeholder: 'Select an outbound to route through',
    searchPlaceholder: 'Type to filter outbounds...',
    noOptions: 'No matching outbounds found',
    none: 'No other outbounds available',
    help: 'Route through another outbound',
  },
  bindInterface: {
    label: 'Bind Interface',
    placeholder: 'e.g., eth0, en0',
    help: 'Bind to specific network interface',
  },
  inet4BindAddress: {
    label: 'IPv4 Bind Address',
    placeholder: 'e.g., 192.168.1.100',
    help: 'Local IPv4 address to bind',
  },
  inet6BindAddress: {
    label: 'IPv6 Bind Address',
    placeholder: 'e.g., ::1',
    help: 'Local IPv6 address to bind',
  },
  domainStrategy: {
    label: 'Domain Strategy',
    help: 'DNS resolution strategy for domains',
    options: {
      default: 'Default',
      preferIpv4: 'Prefer IPv4',
      preferIpv6: 'Prefer IPv6',
      ipv4Only: 'IPv4 Only',
      ipv6Only: 'IPv6 Only',
    },
  },
  advanced: 'Advanced Options',
  connectTimeout: {
    label: 'Connect Timeout',
    placeholder: 'e.g., 5s, 30s',
    help: 'Connection timeout duration',
  },
  fallbackDelay: {
    label: 'Fallback Delay',
    placeholder: 'e.g., 300ms',
    help: 'Delay before fallback',
  },
  protectPath: {
    label: 'Protect Path',
    hint: '(Android)',
    placeholder: 'e.g., /dev/protect',
    help: 'Android VPN protect path',
  },
  routingMark: {
    label: 'Routing Mark',
    hint: '(Linux)',
    placeholder: 'e.g., 255',
    help: 'SO_MARK socket option',
  },
  reuseAddr: {
    label: 'Reuse Address',
    hint: '(SO_REUSEADDR)',
  },
  tcpFastOpen: {
    label: 'TCP Fast Open',
    hint: '(TFO)',
  },
  tcpMultiPath: {
    label: 'TCP Multi-Path',
    hint: '(MPTCP)',
  },
  udpFragment: {
    label: 'UDP Fragment',
    help: 'Control UDP packet fragmentation',
    options: {
      default: 'Default',
      enabled: 'Enabled',
      disabled: 'Disabled',
    },
  },
}
