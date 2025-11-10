# sing-box-easy API v1.13.0 文档

本文档描述了 sing-box-easy v1.13.0 版本的 RESTful API 接口。

## 基本信息

- **Base URL**: `/1.13.0`
- **Content-Type**: `application/json`
- **版本**: 1.13.0 (对应 sing-box 1.13.0)

---

## 一、配置文件管理 API

### 1.1 获取当前配置

**接口**: `GET /1.13.0/config`

**作用**: 获取当前的 sing-box 配置

**响应示例**:
```json
{
  "log": {...},
  "dns": {...},
  "inbounds": [...],
  "outbounds": [...],
  "route": {...}
}
```

---

### 1.2 验证配置

**接口**: `POST /1.13.0/config/validate`

**作用**: 验证配置文件是否合法（使用 sing-box check 命令）

**请求体**: 完整的配置 JSON

**响应示例**:
```json
{
  "valid": true,
  "message": "configuration is valid"
}
```

---

### 1.3 获取备份配置

**接口**: `GET /1.13.0/config/backup`

**作用**: 获取上一次备份的配置文件

**响应**: 备份的配置 JSON

---

### 1.4 回滚配置

**接口**: `POST /1.13.0/config/rollback`

**作用**: 将配置回滚到上一个版本

**响应示例**:
```json
{
  "message": "configuration rolled back successfully"
}
```

---

## 二、节点解析与管理 API

### 2.1 解析节点

**接口**: `POST /1.13.0/nodes/parse`

**作用**: 解析 base64 编码的节点/订阅链接

**请求体**:
```json
{
  "subscription": "节点内容（每行一个，支持多行）"
}
```

**响应示例**:
```json
{
  "message": "nodes parsed successfully",
  "node_count": 10,
  "nodes": [...]
}
```

---

### 2.2 获取所有 Outbound

**接口**: `GET /1.13.0/outbounds`

**作用**: 获取所有 outbound 节点配置

**响应示例**:
```json
{
  "outbounds": [
    {
      "tag": "🇭🇰 香港 01",
      "type": "shadowsocks",
      "server": "example.com",
      "server_port": 8388,
      ...
    }
  ]
}
```

---

### 2.3 获取指定 Outbound

**接口**: `GET /1.13.0/outbounds/:tag`

**作用**: 获取指定 tag 的 outbound 配置

**路径参数**:
- `tag`: outbound 的 tag

**响应**: 单个 outbound 配置对象

---

### 2.4 添加 Outbound

**接口**: `POST /1.13.0/outbounds`

**作用**: 添加新的 outbound 节点

**请求体**:
```json
{
  "tag": "🇯🇵 日本 01",
  "type": "shadowsocks",
  "server": "jp.example.com",
  "server_port": 8388,
  "method": "aes-256-gcm",
  "password": "password"
}
```

**响应示例**:
```json
{
  "message": "outbound added successfully",
  "tag": "🇯🇵 日本 01"
}
```

---

### 2.5 批量添加 Outbound

**接口**: `POST /1.13.0/outbounds/batch`

**作用**: 批量添加多个 outbound 节点（通常用于从订阅解析后批量添加）

**请求体**:
```json
{
  "outbounds": [
    {
      "tag": "🇯🇵 日本 01",
      "type": "shadowsocks",
      "server": "jp1.example.com",
      "server_port": 8388,
      "method": "aes-256-gcm",
      "password": "password"
    },
    {
      "tag": "🇯🇵 日本 02",
      "type": "shadowsocks",
      "server": "jp2.example.com",
      "server_port": 8388,
      "method": "aes-256-gcm",
      "password": "password"
    }
  ]
}
```

**响应示例（全部添加成功）**:
```json
{
  "message": "outbounds batch add completed",
  "added_count": 2,
  "added_tags": ["🇯🇵 日本 01", "🇯🇵 日本 02"]
}
```

**响应示例（部分已存在）**:
```json
{
  "message": "added 1 outbounds, skipped 1 existing outbounds",
  "added_count": 1,
  "added_tags": ["🇯🇵 日本 02"],
  "skipped_count": 1,
  "skipped_tags": ["🇯🇵 日本 01"]
}
```

**说明**:
- 如果某个节点的 tag 已存在，会跳过该节点并继续添加其他节点
- 只要有至少一个节点添加成功，接口就返回成功
- 如果所有节点都已存在，返回错误

---

### 2.6 更新 Outbound

**接口**: `PUT /1.13.0/outbounds/:tag`

**作用**: 更新指定 tag 的 outbound 配置

**路径参数**:
- `tag`: outbound 的 tag

**请求体**: 完整的 outbound 配置对象

---

### 2.6 删除 Outbound

**接口**: `DELETE /1.13.0/outbounds/:tag`

**作用**: 删除指定 tag 的 outbound

**路径参数**:
- `tag`: outbound 的 tag

---

### 2.7 获取所有分组

**接口**: `GET /1.13.0/outbounds/groups`

**作用**: 获取所有分组类型的 outbound（selector/urltest）

**响应示例**:
```json
{
  "groups": [
    {
      "tag": "🚀 节点选择",
      "type": "selector",
      "outbounds": ["节点1", "节点2"]
    }
  ]
}
```

---

### 2.8 更新分组成员

**接口**: `PUT /1.13.0/outbounds/:tag/members`

**作用**: 更新分组的成员列表

**路径参数**:
- `tag`: 分组的 tag

**请求体**:
```json
{
  "outbounds": ["🇭🇰 香港 01", "🇯🇵 日本 01", "🇺🇸 美国 01"]
}
```

---

## 三、DNS 配置 API

### 3.1 获取所有 DNS 服务器

**接口**: `GET /1.13.0/dns/servers`

**作用**: 获取所有 DNS 服务器配置

---

### 3.2 添加 DNS 服务器

**接口**: `POST /1.13.0/dns/servers`

**作用**: 添加新的 DNS 服务器

**请求体**:
```json
{
  "tag": "dns_cloudflare",
  "address": "1.1.1.1",
  "detour": "🚀 节点选择"
}
```

---

### 3.3 获取指定 DNS 服务器

**接口**: `GET /1.13.0/dns/servers/:tag`

---

### 3.4 更新 DNS 服务器

**接口**: `PUT /1.13.0/dns/servers/:tag`

---

### 3.5 删除 DNS 服务器

**接口**: `DELETE /1.13.0/dns/servers/:tag`

---

### 3.6 获取 Hosts 配置

**接口**: `GET /1.13.0/dns/hosts`

**作用**: 获取静态 DNS hosts 配置

**响应示例**:
```json
{
  "hosts": {
    "home.example.com": ["192.168.1.1"],
    "nas.example.com": ["192.168.1.2"]
  }
}
```

---

### 3.7 更新 Hosts 配置

**接口**: `PUT /1.13.0/dns/hosts`

**请求体**: hosts 映射对象

---

### 3.8 获取所有 DNS 规则

**接口**: `GET /1.13.0/dns/rules`

---

### 3.9 添加 DNS 规则

**接口**: `POST /1.13.0/dns/rules`

**请求体**: DNS 规则配置对象

---

### 3.10 更新 DNS 规则

**接口**: `PUT /1.13.0/dns/rules/:index`

**路径参数**:
- `index`: 规则索引（从 0 开始）

---

### 3.11 删除 DNS 规则

**接口**: `DELETE /1.13.0/dns/rules/:index`

---

## 四、Inbound 配置 API

### 4.1 获取所有 Inbound

**接口**: `GET /1.13.0/inbounds`

---

### 4.2 添加 Inbound

**接口**: `POST /1.13.0/inbounds`

**请求体示例**:
```json
{
  "tag": "tun-in",
  "type": "tun",
  "address": ["172.19.0.1/30"],
  "auto_route": true
}
```

---

### 4.3 获取指定 Inbound

**接口**: `GET /1.13.0/inbounds/:tag`

---

### 4.4 更新 Inbound

**接口**: `PUT /1.13.0/inbounds/:tag`

---

### 4.5 删除 Inbound

**接口**: `DELETE /1.13.0/inbounds/:tag`

---

## 五、Route 路由配置 API

### 5.1 获取所有路由规则

**接口**: `GET /1.13.0/route/rules`

---

### 5.2 添加路由规则

**接口**: `POST /1.13.0/route/rules`

**请求体示例**:
```json
{
  "ip_cidr": ["172.17.0.0/16"],
  "outbound": "➡️ 直连"
}
```

---

### 5.3 更新路由规则

**接口**: `PUT /1.13.0/route/rules/:index`

---

### 5.4 删除路由规则

**接口**: `DELETE /1.13.0/route/rules/:index`

---

### 5.5 获取所有规则集

**接口**: `GET /1.13.0/route/rule-sets`

---

### 5.6 添加规则集

**接口**: `POST /1.13.0/route/rule-sets`

**请求体示例**:
```json
{
  "tag": "geosite-custom",
  "type": "remote",
  "format": "binary",
  "url": "https://example.com/custom.srs",
  "download_detour": "➡️ 直连"
}
```

---

### 5.7 获取指定规则集

**接口**: `GET /1.13.0/route/rule-sets/:tag`

---

### 5.8 更新规则集

**接口**: `PUT /1.13.0/route/rule-sets/:tag`

---

### 5.9 删除规则集

**接口**: `DELETE /1.13.0/route/rule-sets/:tag`

---

### 5.10 获取兜底策略

**接口**: `GET /1.13.0/route/final`

**响应示例**:
```json
{
  "final": "🐠 兜底策略"
}
```

---

### 5.11 更新兜底策略

**接口**: `PUT /1.13.0/route/final`

**请求体**:
```json
{
  "final": "➡️ 直连"
}
```

---

## 六、日志配置 API

### 6.1 获取日志配置

**接口**: `GET /1.13.0/log`

---

### 6.2 更新日志配置

**接口**: `PUT /1.13.0/log`

**请求体示例**:
```json
{
  "disabled": false,
  "level": "info",
  "timestamp": true
}
```

---

## 七、实验性功能配置 API

### 7.1 获取 Clash API 配置

**接口**: `GET /1.13.0/experimental/clash-api`

---

### 7.2 更新 Clash API 配置

**接口**: `PUT /1.13.0/experimental/clash-api`

**请求体示例**:
```json
{
  "external_controller": "0.0.0.0:9095",
  "external_ui": "/etc/sing-box/ui",
  "secret": "",
  "default_mode": "rule"
}
```

---

### 7.3 获取缓存文件配置

**接口**: `GET /1.13.0/experimental/cache-file`

---

### 7.4 更新缓存文件配置

**接口**: `PUT /1.13.0/experimental/cache-file`

---

## 八、服务控制 API

### 8.1 获取服务状态

**接口**: `GET /1.13.0/service/status`

**响应示例**:
```json
{
  "status": "running",
  "running": true
}
```

---

### 8.2 启动服务

**接口**: `POST /1.13.0/service/start`

**作用**: 启动 sing-box 服务（启动前会先验证配置）

**响应示例**:
```json
{
  "message": "service started successfully"
}
```

---

### 8.3 停止服务

**接口**: `POST /1.13.0/service/stop`

---

### 8.4 重启服务

**接口**: `POST /1.13.0/service/restart`

**作用**: 重启 sing-box 服务（重启前会先验证配置）

---

## 九、订阅管理 API

### 9.1 获取所有订阅

**接口**: `GET /1.13.0/subscriptions`

**响应示例**:
```json
{
  "subscriptions": [
    {
      "id": "sub_1234567890",
      "name": "我的订阅",
      "url": "https://example.com/sub",
      "auto_update": true,
      "update_interval": "24h",
      "last_update": "2025-01-01T00:00:00Z",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### 9.2 获取指定订阅

**接口**: `GET /1.13.0/subscriptions/:id`

---

### 9.3 添加订阅

**接口**: `POST /1.13.0/subscriptions`

**请求体**:
```json
{
  "name": "我的订阅",
  "url": "https://example.com/subscription",
  "auto_update": true,
  "update_interval": "24h"
}
```

---

### 9.4 更新订阅配置

**接口**: `PUT /1.13.0/subscriptions/:id`

---

### 9.5 删除订阅

**接口**: `DELETE /1.13.0/subscriptions/:id`

---

### 9.6 手动更新订阅内容

**接口**: `POST /1.13.0/subscriptions/:id/update`

**作用**: 手动拉取订阅并解析节点

**响应示例**:
```json
{
  "message": "subscription updated successfully",
  "id": "sub_1234567890",
  "node_count": 50,
  "nodes": [...]
}
```

---

## 配置修改安全机制

所有修改配置的 API（POST/PUT/DELETE）都遵循以下安全流程：

1. 创建临时配置文件 `config_new.json`
2. 使用 `sing-box check` 命令验证配置
3. 如果验证通过：
   - `config.json` → `config.old.json`（备份）
   - `config_new.json` → `config.json`（应用新配置）
4. 如果验证失败：
   - 保持原配置不变
   - 删除 `config_new.json`
   - 返回错误信息

这确保了配置修改的安全性，任何时候都可以通过 `/config/rollback` 回滚到上一个正确的配置。

---

## 错误响应格式

所有 API 在发生错误时返回以下格式：

```json
{
  "error": "错误描述信息"
}
```

HTTP 状态码：
- `200 OK`: 成功
- `201 Created`: 创建成功
- `400 Bad Request`: 请求参数错误
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误
