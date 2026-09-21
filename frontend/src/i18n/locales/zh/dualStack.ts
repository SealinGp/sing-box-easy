export default {
  title: 'IPv4 / IPv6 域名测试',
  desc: '按 sing-box 的方式解析域名，然后向每个地址族发送真实流量，并报告实际命中的规则。',

  domains: '域名',
  domainsHint: '每行一个，或用逗号分隔，最多 {max} 个。',
  placeholder: 'www.baidu.com\nwww.google.com',
  run: '开始测试',
  running: '测试中…',
  hint: '输入一个或多个域名，同时测试两个地址族。',

  showAdvanced: '显示选项',
  hideAdvanced: '隐藏选项',
  fields: {
    port: '端口',
    tls: '完成 TLS 握手',
    skipDial: '不发送流量',
    timeout: '单次连接超时（毫秒）',
  },
  tlsHelp: '可验证链路确实到达目标服务器，而不仅仅是某个在监听的东西。',
  skipDialHelp: '仅报告 DNS 结果与路由预测。',

  stage: {
    endpoint: '正在读取配置…',
    resolve: '正在通过 sing-box 解析…',
    predict: '正在走一遍路由规则…',
    traffic: '正在发送流量…',
  },

  endpoint: {
    title: '客户端 DNS',
    inbound: '由入站 {inbound}（{address}）劫持',
    protocol_dns: '由 protocol=dns 规则在所有入站上劫持 —— 没有单一地址可供客户端指向',
    none: '没有任何路由规则劫持 DNS，客户端不会经过 sing-box 解析',
    unresolved: '入站 {inbound} 未设置监听端口，因此没有可查询的地址',
  },

  client: {
    noIpv6: '本机没有全局 IPv6 地址，因此 IPv6 行未做测试。',
    noIpv4: '本机没有全局 IPv4 地址，因此 IPv4 行未做测试。',
  },

  family: {
    ipv4: 'IPv4',
    ipv6: 'IPv6',
  },

  status: {
    reachable: '可达',
    unreachable: '不可达',
    no_address: '无地址',
    untested: '未测试',
  },
  statusHelp: {
    no_address: 'sing-box 未返回该类型记录。对于走代理的域名，这正是 IPv6 分流应有的结果。',
    untested: '没有发起连接，因此无法判断链路是否可用。',
  },

  agreement: {
    agree: '与预测一致',
    differ: '与预测不一致',
    unknown: '未验证',
  },
  differHelp: '配置预测为 {predicted}，但流量实际从 {observed} 出去。',
  inexact: '该预测并不精确 —— 判定之前存在无法确定的规则 —— 因此这里的不一致可能是预测的问题，而非配置的问题。',

  predicted: '预测',
  observed: '实际',
  via: '经由',
  noObservation: 'sing-box 中未出现匹配的连接',
  observeUnavailable: '无法读取连接信息，因此无法确认：{error}',

  toast: {
    failed: '测试失败',
  },
}
