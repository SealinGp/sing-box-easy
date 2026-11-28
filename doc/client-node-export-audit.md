# 入站到客户端出站的协议核对（2026-11-28）

核对来源：本地 sing-box 仓库的 v1.14.1 标签及 option/、protocol/ 下的实现。
面板目前编译依赖 v1.12.12；生成结果另用官方 1.14.1 二进制检查。

| 入站 | 客户端映射 | 处理 |
| --- | --- | --- |
| VMess | users.uuid；alterId → alter_id；security=auto | name 仅是服务端日志/路由用户标识，不导出 username |
| VLESS | users.uuid、flow | 保留 flow，不导出 name/username |
| Trojan / AnyTLS | users.password | 不导出显示名称 |
| TUIC | users.uuid + password | 同时保留两项，保留拥塞/握手参数 |
| Hysteria | users.auth / auth_str | 保留混淆，服务端带宽方向转换为客户端方向 |
| Hysteria2 | users.password | 保留 obfs，不复制 masquerade/服务端带宽限制 |
| Shadowsocks | method + password | 2022 多用户密码为 server-key:user-key；托管/中继需单独配置 |
| SOCKS / HTTP | users.username + password | username 是真正的认证字段，必须保留 |
| Mixed | SOCKS；启用 TLS 时使用 HTTP | 不把 TLS 配给不支持它的 SOCKS 出站 |
| ShadowTLS | 包装层，需要配合内部代理 | 明确提示无法作为单个完整节点导出 |
| Naive | 1.14 客户端有相应协议，但面板当前 registry 无此出站 | 明确提示当前导入不支持 |
| TUN / redirect / tproxy / direct | 本地/透明流量入口 | 不生成伪远程代理 |

TLS 只复制客户端字段，保留 SNI 和公开证书（若原配置直接提供）；不复制服务端文件路径、私钥、ACME、证书提供器。Reality 通过 X25519 从私钥推导公共密钥，取 short_id，保留握手 SNI，客户端启用 uTLS。ECH 和客户端证书认证需要额外客户端材料，明确阻止不完整导出。

多用户在复制面板内选择；公网地址可修改。NAT 公网端口需要用户核对，不擅自推断。

原生 JSON 添加保留未知原生选项供后端校验，仅兼容清理旧复制功能错误生成的非用户名认证协议 username。批量导入也支持单对象、数组、完整 outbounds 集合。用途模板只接受独立叶代理节点，拒绝带组或 detour/resolver 引用的输入，避免重命名后悬空引用。

验证包含各协议认证字段、已发布 1.14.1 核心检查、Reality 私钥隔离、无认证信息拒绝、用户选择、无效 JSON，以及浏览器粘贴/编辑/错误恢复。核心配置检查并不替代对真实出口服务器的连通性测试。
