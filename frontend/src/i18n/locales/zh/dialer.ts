// 共享拨号选项区块（DialerOptions.vue）。
export default {
  title: '拨号选项',
  optional: '（可选）',
  detour: {
    label: '前置代理',
    hint: '（用于链式代理的前置出站标签）',
    placeholder: '选择一个要经过的出站',
    searchPlaceholder: '输入以筛选出站...',
    noOptions: '未找到匹配的出站',
    none: '没有其他可用的出站',
    help: '通过另一个出站进行路由',
  },
  bindInterface: {
    label: '绑定网卡',
    placeholder: '例如 eth0、en0',
    help: '绑定到指定的网络接口',
  },
  inet4BindAddress: {
    label: 'IPv4 绑定地址',
    placeholder: '例如 192.168.1.100',
    help: '要绑定的本地 IPv4 地址',
  },
  inet6BindAddress: {
    label: 'IPv6 绑定地址',
    placeholder: '例如 ::1',
    help: '要绑定的本地 IPv6 地址',
  },
  domainStrategy: {
    label: '域名策略',
    help: '域名的 DNS 解析策略',
    options: {
      default: '默认',
      preferIpv4: '优先 IPv4',
      preferIpv6: '优先 IPv6',
      ipv4Only: '仅 IPv4',
      ipv6Only: '仅 IPv6',
    },
  },
  advanced: '高级选项',
  connectTimeout: {
    label: '连接超时',
    placeholder: '例如 5s、30s',
    help: '连接超时时长',
  },
  fallbackDelay: {
    label: '回退延迟',
    placeholder: '例如 300ms',
    help: '回退前的延迟',
  },
  protectPath: {
    label: 'Protect 路径',
    hint: '（Android）',
    placeholder: '例如 /dev/protect',
    help: 'Android VPN protect 路径',
  },
  routingMark: {
    label: '路由标记',
    hint: '（Linux）',
    placeholder: '例如 255',
    help: 'SO_MARK 套接字选项',
  },
  reuseAddr: {
    label: '复用地址',
    hint: '（SO_REUSEADDR）',
  },
  tcpFastOpen: {
    label: 'TCP Fast Open',
    hint: '（TFO）',
  },
  tcpMultiPath: {
    label: 'TCP 多路径',
    hint: '（MPTCP）',
  },
  udpFragment: {
    label: 'UDP 分片',
    help: '控制 UDP 数据包分片',
    options: {
      default: '默认',
      enabled: '启用',
      disabled: '禁用',
    },
  },
}
