// 入站页（Inbounds.vue）。
export default {
  title: '入站管理',
  add: '添加入站',
  addFirst: '添加第一个入站',
  empty: '尚未配置入站',
  table: {
    tag: '标签',
    type: '类型',
    listenAddress: '监听地址',
    port: '端口',
    sniff: '探测',
    actions: '操作',
  },
  sniff: {
    enabled: '已启用',
    disabled: '已禁用',
  },
  modal: {
    edit: '编辑入站',
    add: '添加入站',
  },
  form: {
    tag: '标签 *',
    tagPlaceholder: '例如：mixed-in',
    tagHelp: '该入站的唯一标识',
    type: '类型 *',
    typePlaceholder: '选择入站类型',
    generate: '重新生成',
    enabled: '启用',
    addField: '添加字段：',
    addDeprecatedField: '已弃用：',
    deprecatedHint: '在 sing-box 1.12 中已弃用 —— 仍可使用，但新配置不应再使用该字段。',
    addUser: '添加用户',
    invalidJson: 'JSON 格式错误：{message}',
    flowNone: '无',

    /*
     * 仅为「名字本身讲不清楚」的字段提供中文名。
     *
     * 这里刻意不求全：生成的字段清单共约 250 个字段名，未列出的会在渲染时
     * 由字段名自动转换（"tcp_fast_open" → "TCP Fast Open"），因此 sing-box
     * 新增字段无需改动本文件即可正常显示。
     */
    fields: {
      listen: '监听地址',
      listen_port: '监听端口',
      users: '用户',
      method: '加密方法',
      password: '密码',
      set_system_proxy: '设为系统代理',
      detour: '出站绕行',
      network: '网络',
      tls: 'TLS',
      transport: '传输层',
      multiplex: '多路复用',
      domain_resolver: '域名解析器',
      // tun
      address: '接口地址',
      auto_route: '自动路由',
      strict_route: '严格路由',
      stack: '网络栈',
      route_address: '路由地址',
      route_exclude_address: '排除路由地址',
      // shadowsocks
      destinations: '中继目标',
      managed: '由 SSM API 管理',
      // direct
      override_address: '覆盖地址',
      override_port: '覆盖端口',
      // hysteria / hysteria2
      up_mbps: '上行带宽 (Mbps)',
      down_mbps: '下行带宽 (Mbps)',
      // tuic
      congestion_control: '拥塞控制',
      zero_rtt_handshake: '零 RTT 握手',
      // shadowtls
      version: 'ShadowTLS 版本',
      handshake: '握手服务器',
      wildcard_sni: '通配符 SNI',
      // anytls
      padding_scheme: '填充方案',
      // trojan
      fallback: '回落',
      fallback_for_alpn: 'ALPN 回落',
    },

    /** users 数组中单个条目的子字段。 */
    userFields: {
      username: '用户名',
      password: '密码',
      name: '名称',
      uuid: 'UUID',
      alterId: 'Alter ID',
      flow: 'Flow',
      auth_str: '认证字符串',
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
    title: '删除入站',
    confirm: '确定要删除入站 {tag} 吗？此操作无法撤销。',
  },
  toast: {
    fetchFailed: '获取入站列表失败',
    addedOk: '入站添加成功',
    updatedOk: '入站更新成功',
    saveFailed: '保存入站失败',
    deletedOk: '入站删除成功',
    deleteFailed: '删除入站失败',
    copiedOk: '客户端出站配置已复制到剪贴板',
    copyFailed: '复制配置失败',
  },
  tooltip: {
    copyConfig: '复制客户端出站配置',
  },
  validation: {
    title: '校验错误',
    tagRequired: '标签为必填项',
    typeRequired: '类型为必填项',
    listenPortRequired: '监听端口为必填项',
    ssMethodRequired: '加密方法为必填项',
    ssPasswordRequired: '密码为必填项',
    usersRequired: '该入站类型至少需要一个用户',
    userIdentityRequired: '每个用户都需要填写凭据（UUID 或密码）',
  },
}
