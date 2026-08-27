// DNS pages (DNS.vue, DNSServers.vue, DNSRules.vue, DNSSettings.vue).
export default {
  title: 'DNS Configuration',
  tabs: {
    servers: 'DNS Servers',
    rules: 'DNS Rules',
    settings: 'Settings',
    diagnostics: 'Diagnostics',
  },
  servers: {
    heading: 'DNS Servers',
    add: 'Add DNS Server',
    addFirst: 'Add Your First DNS Server',
    empty: 'No DNS servers configured',
    table: {
      tag: 'Tag',
      type: 'Type',
      server: 'Server',
      port: 'Port',
      detour: 'Detour',
      actions: 'Actions',
    },
    types: {
      udp: 'UDP',
      tcp: 'TCP',
      tls: 'DNS over TLS',
      https: 'DNS over HTTPS',
      h3: 'DNS over HTTP/3',
      tailscale: 'Tailscale',
      quic: 'DNS over QUIC',
      local: 'Local System DNS',
      dhcp: 'DHCP',
      fakeip: 'FakeIP',
      hosts: 'Hosts File',
    },
    hostsSummary: {
      predefined: '{n} predefined',
      file: '{n} file',
      files: '{n} files',
      empty: '(empty)',
    },
    modal: {
      add: 'Add DNS Server',
      edit: 'Edit DNS Server',
    },
    form: {
      tag: 'Tag *',
      tagPlaceholder: 'e.g., cloudflare',
      type: 'Type *',
      typePlaceholder: 'Pick dns type',
      serverAddress: 'Server Address *',
      serverAddressPlaceholder: '1.1.1.1 or dns.cloudflare.com',
      port: 'Port',
      portPlaceholder: '53',
      path: 'Path',
      pathPlaceholder: '/dns-query',
      detour: 'Detour (outbound)',
      detourDirect: 'Direct (no proxy)',
      detourHelp: 'Send this resolver\u2019s queries through an outbound. Leave direct for a resolver that must work without the proxy \u2014 in particular the one named by route.default_domain_resolver, which resolves the proxy servers\u2019 own hostnames.',
      pathHelp: 'URL path for DoH queries',
      interface: 'Interface',
      interfacePlaceholder: 'e.g., eth0',
      inet4Range: 'IPv4 Range',
      inet4RangePlaceholder: '198.18.0.0/15',
      inet6Range: 'IPv6 Range',
      inet6RangePlaceholder: 'fc00::/18',
      predefinedHosts: 'Predefined Hosts',
      predefinedHostsHelp: 'One mapping per line: {format}. Lines starting with {hash} and blank lines are ignored.',
      hostsFilePaths: 'Hosts File Paths',
      hostsFilePathsOptional: '(optional)',
      hostsFilePathsPlaceholder: '/etc/hosts',
      hostsFilePathsHelp: 'One path per line. Loaded in addition to the predefined mappings above.',

      // ââ schema-driven form ââââââââââââââââââââââââââââââââââââââââââââââââ
      localHint:
        'A local resolver uses the host’s own DNS settings, so it has no address to configure. Everything below is an optional dialer setting.',
      addHost: 'Add host',
      hostsDomainPlaceholder: 'example.com',
      hostsAddressPlaceholder: '192.0.2.1',
      hostsHelp: 'Separate multiple addresses for one domain with commas.',

      /*
       * Labels for fields whose sing-box name is not self-explanatory.
       *
       * Deliberately NOT exhaustive — anything absent is humanized from its own
       * key at render time ('tcp_fast_open' → 'TCP Fast Open'), so a field added
       * by a future sing-box stays legible without touching this file.
       */
      fields: {
        server: 'Server Address',
        server_port: 'Port',
        detour: 'Detour (outbound)',
        path: 'Path',
        method: 'HTTP Method',
        headers: 'HTTP Headers',
        tls: 'TLS',
        domain_resolver: 'Domain Resolver',
        connect_timeout: 'Connect Timeout',
        fallback_delay: 'Fallback Delay',
        interface: 'Interface',
        inet4_range: 'IPv4 Range',
        inet6_range: 'IPv6 Range',
        predefined: 'Predefined Hosts',
        endpoint: 'Tailscale Endpoint',
        accept_default_resolvers: 'Accept Default Resolvers',
      },
    },
    validation: {
      tagRequired: 'Tag is required',
      typeRequired: 'Type is required',
      serverRequired: 'Server address is required for this DNS type',
    },
    del: {
      title: 'Delete DNS Server',
      confirm: 'Are you sure you want to delete the DNS server {tag}? This action cannot be undone.',
    },
    toast: {
      addedOk: 'DNS server added successfully',
      updatedOk: 'DNS server updated successfully',
      saveFailed: 'Failed to save DNS server',
      deletedOk: 'DNS server deleted successfully',
      deleteFailed: 'Failed to delete DNS server',
    },
  },
  rules: {
    heading: 'DNS Rules',
    subheading: 'Rules are processed in order. First match wins.',
    add: 'Add DNS Rule',
    addFirst: 'Add Your First DNS Rule',
    empty: 'No DNS rules configured',
    table: {
      index: '#',
      action: 'Action',
      server: 'Server',
      conditions: 'Conditions',
      actions: 'Actions',
      answerCount: '{count} records',
    },
    actionTypes: {
      route: 'Route - Forward to specified DNS server',
      routeOptions: 'Route Options - Set route options without changing server',
      reject: 'Reject - Reject DNS requests with specific method',
      predefined: 'Predefined - Answer the query directly with a fixed response code',
    },
    rejectMethods: {
      default: 'Default - Return an empty response',
      drop: 'Drop - Send no response at all',
    },
    summary: {
      // The per-field labels used to live here. The conditions cell now
      // resolves them through `dns.rules.form.fields.*` (humanized from the
      // JSON key when untranslated), shared with the flow preview — so a
      // condition this form has no control for is still named correctly.
      none: 'No conditions',
    },
    modal: {
      add: 'Add DNS Rule',
      edit: 'Edit DNS Rule',
    },
    flow: {
      then: {
        route: 'resolve it with {server}',
        routeIncomplete: 'resolve it with \u2026 (pick a DNS server)',
        routeOptions: 'apply these resolution options',
        reject: 'refuse it (empty response)',
        rejectDrop: 'drop it silently',
        predefined: 'answer it directly',
        predefinedRcode: 'answer it directly with {rcode}',
      },
    },
    form: {
      whenHeading: 'When a DNS query matches',
      whenHint: 'all of the conditions below',
      thenHeading: 'Then',
      action: 'Action *',
      server: 'DNS Server *',
      rejectMethod: 'Reject Method',
      rcode: 'Response code',
      rcodeHelp: 'Returned to the client instead of querying a server. NXDOMAIN is the usual choice for blocking.',
      rejectMethodHelp: 'Method to reject DNS requests',
      // Labels for the schema-driven action fields. Only fields where a human
      // label beats the JSON key need an entry — anything absent falls back to a
      // humanized key ("disable_cache" -> "Disable Cache"), which is why
      // `strategy` and `disable_cache` are not listed.
      fields: {
        server: 'DNS Server',
        // Condition labels, for the flow preview. The conditions half of this
        // form is still hand-written, so these are not generated — but the
        // preview resolves them the same way, `<prefix>.<json key>`.
        rule_set: 'Rule set',
        domain: 'Domain',
        domain_suffix: 'Domain suffix',
        domain_keyword: 'Domain keyword',
        geosite: 'GeoSite (removed in 1.12)',
        rewrite_ttl: 'Rewrite TTL',
        client_subnet: 'Client Subnet (EDNS)',
        method: 'Reject Method',
        no_drop: 'Do Not Drop',
        noDropHint: 'Only valid with the "default" method — sing-box rejects it alongside "drop".',
        rcode: 'Response Code',
        answer: 'Answer Records',
        answerHint: 'One DNS resource record per entry, e.g. "example.com. 3600 IN A 192.0.2.1".',
        ns: 'Authority Records',
        extra: 'Additional Records',
      },
      errors: {
        serverRequired: 'A route action needs a DNS server to route to.',
        routeOptionsEmpty:
          'A route-options action must set at least one option — sing-box rejects an empty one.',
        noDropWithDrop: '"Do Not Drop" cannot be combined with the "drop" method.',
      },
      conditionsHeading: 'Conditions (at least one required)',
      ruleSet: 'Rule Set',
      ruleSetSelect: 'Select or search a rule set',
      ruleSetSearch: 'Type to filter rule sets...',
      ruleSetNoOptions: 'No matching rule sets found',
      ruleSetMissing: '{tag} — not configured',
      ruleSetEmpty: 'No rule sets configured yet. Add one under Route → Rule Sets first.',
      ruleSetHelp: 'Use a predefined rule set for this DNS rule',
      // Rule-set vs domain-condition guidance. sing-box ANDs the matchers
      // inside one rule, so mixing the two narrows the scope instead of
      // widening it — see components/DNSRuleConditions.vue.
      // The "add a matcher" row under the domain conditions — see
      // components/DNSRuleConditions.vue.
      matchers: {
        add: 'Add a condition:',
      },
      mixing: {
        title: 'Conditions are combined with AND',
        warning:
          'A rule set and domain conditions in the same rule must BOTH match: only a domain that is in the rule set AND matches the domain condition hits this rule. To let every domain in the rule set through, leave the domain conditions empty — and vice versa. Use a second rule if you meant "either one".',
        ruleSetCollapsed: 'This rule matches by domain conditions. Rule set hidden.',
        matchersCollapsed: 'This rule matches by rule set. Domain conditions hidden.',
        show: 'Show anyway',
        hide: 'Hide',
        matchersGroup: 'Domain conditions',
      },
      domain: 'Domain',
      domainPlaceholder: 'Add domains (press Enter after each)',
      domainHelp: 'Exact domain match',
      domainSuffix: 'Domain Suffix',
      domainSuffixPlaceholder: 'Add suffixes (.example.com)',
      domainSuffixHelp: 'Matches domain and all subdomains',
      domainKeyword: 'Domain Keyword',
      domainKeywordPlaceholder: 'Add keywords',
      domainKeywordHelp: 'Domain contains keyword',
      geosite: 'GeoSite',
      geositePlaceholder: 'Add geosite tags (e.g. google, netflix, cn)',
      geositeHelp: 'Use geosite database',
    },
    serverSelect: 'Select a server',
    ruleSetLabel: '{tag} ({type} - {format})',
    del: {
      title: 'Delete DNS Rule',
      confirm: 'Are you sure you want to delete rule #{index}? This action cannot be undone.',
    },
    toast: {
      fetchFailed: 'Failed to fetch DNS rules',
      addedOk: 'DNS rule added successfully',
      updatedOk: 'DNS rule updated successfully',
      saveFailed: 'Failed to save DNS rule',
      deletedOk: 'DNS rule deleted successfully',
      deleteFailed: 'Failed to delete DNS rule',
      reordered: 'Rule order saved',
      reorderFailed: 'Failed to save the new rule order',
    },
    reorder: {
      start: 'Reorder',
      done: 'Done',
      save: 'Save order',
      cancel: 'Cancel',
      hint: 'Drag rules by the handle to reorder them, then save. Rules are matched top-down — the first match wins.',
      dirtyHint: 'Order changed — not saved yet.',
      handle: 'Drag to reorder rule {position}; use the arrow keys to move it one place',
    },
  },
  settings: {
    heading: 'Global DNS Settings',
    strategy: 'Domain Strategy',
    strategyHelp: 'IP version preference for DNS queries',
    strategyOptions: {
      preferIpv4: 'Prefer IPv4',
      preferIpv6: 'Prefer IPv6',
      ipv4Only: 'IPv4 Only',
      ipv6Only: 'IPv6 Only',
    },
    finalServer: 'Final DNS Server',
    finalServerHelp: 'Fallback server when no rules match',
    finalServerSelect: 'Select default server',
    cacheHeading: 'Cache Settings',
    disableCache: 'Disable DNS Cache',
    disableCacheHelp: 'Turn off caching of DNS responses',
    disableExpire: 'Disable Cache Expiration',
    disableExpireHelp: 'Keep cached DNS records indefinitely',
    save: 'Save Settings',
    toast: {
      fetchFailed: 'Failed to fetch DNS config',
      updatedOk: 'DNS settings updated successfully',
      updateFailed: 'Failed to update DNS settings',
    },
  },
}
