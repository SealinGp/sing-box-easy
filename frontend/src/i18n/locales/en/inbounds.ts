// Inbounds page (Inbounds.vue).
export default {
  title: 'Inbounds Management',
  add: 'Add Inbound',
  addFirst: 'Add Your First Inbound',
  empty: 'No inbounds configured',
  table: {
    tag: 'Tag',
    type: 'Type',
    listenAddress: 'Listen Address',
    port: 'Port',
    sniff: 'Sniff',
    actions: 'Actions',
  },
  sniff: {
    enabled: 'Enabled',
    disabled: 'Disabled',
  },
  modal: {
    edit: 'Edit Inbound',
    add: 'Add Inbound',
  },
  form: {
    tag: 'Tag *',
    tagPlaceholder: 'e.g., mixed-in',
    tagHelp: 'Unique identifier for this inbound',
    type: 'Type *',
    typePlaceholder: 'Pick inbound type',
    generate: 'Regenerate',
    enabled: 'Enabled',
    addField: 'Add field:',
    addDeprecatedField: 'Deprecated:',
    deprecatedHint:
      'Deprecated in sing-box 1.12 — still accepted, but new configs should not use it.',
    addUser: 'Add user',
    invalidJson: 'Invalid JSON: {message}',
    flowNone: 'None',

    /*
     * Labels for fields whose sing-box name is not self-explanatory.
     *
     * Deliberately NOT exhaustive. The generated inventory carries ~250 field
     * names; anything absent here is humanized from its own key at render time
     * ("tcp_fast_open" → "TCP Fast Open"), so a field added by a future
     * sing-box is legible without touching this file. Add an entry only when a
     * human label says more than the field name does.
     */
    fields: {
      listen: 'Listen Address',
      listen_port: 'Listen Port',
      users: 'Users',
      method: 'Encryption Method',
      password: 'Password',
      set_system_proxy: 'Set as System Proxy',
      detour: 'Detour Outbound',
      network: 'Network',
      tls: 'TLS',
      transport: 'Transport',
      multiplex: 'Multiplex',
      domain_resolver: 'Domain Resolver',
      // tun
      address: 'Interface Address',
      auto_route: 'Auto Route',
      strict_route: 'Strict Route',
      stack: 'Network Stack',
      route_address: 'Routed Addresses',
      route_exclude_address: 'Excluded Addresses',
      // shadowsocks
      destinations: 'Relay Destinations',
      managed: 'Managed by SSM API',
      // direct
      override_address: 'Override Address',
      override_port: 'Override Port',
      // hysteria / hysteria2
      up_mbps: 'Upload Bandwidth (Mbps)',
      down_mbps: 'Download Bandwidth (Mbps)',
      // tuic
      congestion_control: 'Congestion Control',
      zero_rtt_handshake: 'Zero-RTT Handshake',
      // shadowtls
      version: 'ShadowTLS Version',
      handshake: 'Handshake Server',
      wildcard_sni: 'Wildcard SNI',
      // anytls
      padding_scheme: 'Padding Scheme',
      // trojan
      fallback: 'Fallback',
      fallback_for_alpn: 'Fallback for ALPN',
    },

    /** Sub-fields of one entry in a `users` array. */
    userFields: {
      username: 'Username',
      password: 'Password',
      name: 'Name',
      uuid: 'UUID',
      alterId: 'Alter ID',
      flow: 'Flow',
      auth_str: 'Auth String',
    },
  },
  types: {
    mixed: 'Mixed (HTTP/SOCKS)',
    http: 'HTTP',
    socks: 'SOCKS',
    tun: 'TUN',
    redirect: 'Redirect',
    tproxy: 'TProxy',
    direct: 'Direct',
    shadowsocks: 'Shadowsocks',
    vmess: 'VMess',
    trojan: 'Trojan',
    vless: 'VLESS',
    hysteria: 'Hysteria',
    hysteria2: 'Hysteria2',
    tuic: 'TUIC',
    naive: 'Naive',
    shadowtls: 'ShadowTLS',
    anytls: 'AnyTLS',
  },
  del: {
    title: 'Delete Inbound',
    confirm: 'Are you sure you want to delete the inbound {tag}? This action cannot be undone.',
  },
  toast: {
    fetchFailed: 'Failed to fetch inbounds',
    addedOk: 'Inbound added successfully',
    updatedOk: 'Inbound updated successfully',
    saveFailed: 'Failed to save inbound',
    deletedOk: 'Inbound deleted successfully',
    deleteFailed: 'Failed to delete inbound',
    copiedOk: 'Client outbound configuration copied to clipboard',
    copyFailed: 'Failed to copy configuration',
  },
  tooltip: {
    copyConfig: 'Copy client outbound config',
  },
  validation: {
    title: 'Validation Error',
    tagRequired: 'Tag is required',
    typeRequired: 'Type is required',
    listenPortRequired: 'Listen port is required',
    ssMethodRequired: 'Encryption method is required',
    ssPasswordRequired: 'Password is required',
    usersRequired: 'At least one user is required for this inbound type',
    userIdentityRequired: 'Every user needs its credential filled in (UUID or password)',
  },
}
