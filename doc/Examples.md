# sing-box-easy 使用示例

## 完整工作流程示例

### 场景一：从订阅链接解析并添加节点

#### 步骤 1: 解析订阅节点

```bash
curl -X POST http://localhost:8080/1.13.0/nodes/parse \
  -H "Content-Type: application/json" \
  -d '{
    "subscription": "ss://YWVzLTEyOC1nY206cGFzc3dvcmQ=@192.168.1.1:8388#节点1\nss://YWVzLTEyOC1nY206cGFzc3dvcmQ=@192.168.1.2:8388#节点2"
  }'
```

响应示例：
```json
{
  "message": "nodes parsed successfully",
  "node_count": 2,
  "nodes": [
    {
      "tag": "节点1",
      "type": "shadowsocks",
      "server": "192.168.1.1",
      "server_port": 8388,
      "method": "aes-128-gcm",
      "password": "password"
    },
    {
      "tag": "节点2",
      "type": "shadowsocks",
      "server": "192.168.1.2",
      "server_port": 8388,
      "method": "aes-128-gcm",
      "password": "password"
    }
  ]
}
```

#### 步骤 2: 批量添加解析的节点

```bash
curl -X POST http://localhost:8080/1.13.0/outbounds/batch \
  -H "Content-Type: application/json" \
  -d '{
    "outbounds": [
      {
        "tag": "节点1",
        "type": "shadowsocks",
        "server": "192.168.1.1",
        "server_port": 8388,
        "method": "aes-128-gcm",
        "password": "password"
      },
      {
        "tag": "节点2",
        "type": "shadowsocks",
        "server": "192.168.1.2",
        "server_port": 8388,
        "method": "aes-128-gcm",
        "password": "password"
      }
    ]
  }'
```

响应示例：
```json
{
  "message": "outbounds batch add completed",
  "added_count": 2,
  "added_tags": ["节点1", "节点2"]
}
```

#### 步骤 3: 验证节点已添加

```bash
curl http://localhost:8080/1.13.0/outbounds
```

---

### 场景二：添加订阅并自动更新节点

#### 步骤 1: 添加订阅

```bash
curl -X POST http://localhost:8080/1.13.0/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "我的订阅",
    "url": "https://example.com/subscription",
    "auto_update": true,
    "update_interval": "24h"
  }'
```

响应：
```json
{
  "message": "subscription added successfully",
  "id": "sub_1234567890"
}
```

#### 步骤 2: 手动更新订阅获取节点

```bash
curl -X POST http://localhost:8080/1.13.0/subscriptions/sub_1234567890/update
```

响应：
```json
{
  "message": "subscription updated successfully",
  "id": "sub_1234567890",
  "node_count": 50,
  "nodes": [...]
}
```

#### 步骤 3: 批量添加订阅节点

使用上一步返回的 nodes 数组，批量添加到配置：

```bash
curl -X POST http://localhost:8080/1.13.0/outbounds/batch \
  -H "Content-Type: application/json" \
  -d '{
    "outbounds": [
      // 从订阅更新接口返回的 nodes 数组
    ]
  }'
```

---

### 场景三：配置代理分组

#### 步骤 1: 获取所有节点

```bash
curl http://localhost:8080/1.13.0/outbounds
```

#### 步骤 2: 创建或更新分组节点

```bash
curl -X POST http://localhost:8080/1.13.0/outbounds \
  -H "Content-Type: application/json" \
  -d '{
    "tag": "🚀 节点选择",
    "type": "selector",
    "outbounds": ["节点1", "节点2", "节点3"]
  }'
```

或者更新现有分组的成员：

```bash
curl -X PUT http://localhost:8080/1.13.0/outbounds/🚀%20节点选择/members \
  -H "Content-Type: application/json" \
  -d '{
    "outbounds": ["节点1", "节点2", "节点3", "节点4"]
  }'
```

---

### 场景四：配置 DNS 分流

#### 步骤 1: 添加 DNS 服务器

```bash
# 添加直连 DNS
curl -X POST http://localhost:8080/1.13.0/dns/servers \
  -H "Content-Type: application/json" \
  -d '{
    "tag": "dns_direct",
    "address": "223.5.5.5",
    "detour": "➡️ 直连"
  }'

# 添加代理 DNS
curl -X POST http://localhost:8080/1.13.0/dns/servers \
  -H "Content-Type: application/json" \
  -d '{
    "tag": "dns_proxy",
    "address": "tls://8.8.8.8",
    "detour": "🚀 节点选择"
  }'
```

#### 步骤 2: 配置 DNS 规则

```bash
curl -X POST http://localhost:8080/1.13.0/dns/rules \
  -H "Content-Type: application/json" \
  -d '{
    "rule_set": "geosite-cn",
    "server": "dns_direct"
  }'
```

#### 步骤 3: 配置静态 hosts

```bash
curl -X PUT http://localhost:8080/1.13.0/dns/hosts \
  -H "Content-Type: application/json" \
  -d '{
    "home.example.com": ["192.168.1.100"],
    "nas.example.com": ["192.168.1.200"]
  }'
```

---

### 场景五：配置路由规则

#### 步骤 1: 添加直连规则

```bash
curl -X POST http://localhost:8080/1.13.0/route/rules \
  -H "Content-Type: application/json" \
  -d '{
    "ip_cidr": ["192.168.0.0/16", "10.0.0.0/8"],
    "outbound": "➡️ 直连"
  }'
```

#### 步骤 2: 添加代理规则

```bash
curl -X POST http://localhost:8080/1.13.0/route/rules \
  -H "Content-Type: application/json" \
  -d '{
    "rule_set": ["geosite-google", "geosite-youtube"],
    "outbound": "🚀 节点选择"
  }'
```

#### 步骤 3: 添加自定义规则集

```bash
curl -X POST http://localhost:8080/1.13.0/route/rule-sets \
  -H "Content-Type: application/json" \
  -d '{
    "tag": "my-custom-rules",
    "type": "remote",
    "format": "binary",
    "url": "https://example.com/custom-rules.srs",
    "download_detour": "➡️ 直连"
  }'
```

---

### 场景六：服务管理

#### 验证配置

在修改配置后，可以先验证配置是否正确：

```bash
curl -X POST http://localhost:8080/1.13.0/config/validate \
  -H "Content-Type: application/json" \
  -d @current-config.json
```

#### 重启服务应用配置

```bash
curl -X POST http://localhost:8080/1.13.0/service/restart
```

#### 查看服务状态

```bash
curl http://localhost:8080/1.13.0/service/status
```

#### 配置回滚

如果发现配置有问题，可以回滚到上一个版本：

```bash
curl -X POST http://localhost:8080/1.13.0/config/rollback
```

---

### 场景七：完整的自动化脚本示例

以下是一个完整的 bash 脚本，实现从订阅更新到应用配置的全流程：

```bash
#!/bin/bash

API_BASE="http://localhost:8080/1.13.0"

# 1. 更新订阅获取节点
echo "正在获取订阅节点..."
SUBSCRIPTION_ID="sub_1234567890"
NODES_RESPONSE=$(curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/update")

# 2. 提取节点列表
NODES=$(echo $NODES_RESPONSE | jq '.nodes')
NODE_COUNT=$(echo $NODES_RESPONSE | jq '.node_count')

echo "获取到 $NODE_COUNT 个节点"

# 3. 批量添加节点
echo "正在批量添加节点..."
ADD_RESPONSE=$(curl -s -X POST "$API_BASE/outbounds/batch" \
  -H "Content-Type: application/json" \
  -d "{\"outbounds\": $NODES}")

ADDED_COUNT=$(echo $ADD_RESPONSE | jq '.added_count')
echo "成功添加 $ADDED_COUNT 个节点"

# 4. 更新分组节点列表
echo "正在更新节点分组..."
NODE_TAGS=$(echo $NODES | jq '[.[].tag]')

curl -s -X PUT "$API_BASE/outbounds/🚀%20节点选择/members" \
  -H "Content-Type: application/json" \
  -d "{\"outbounds\": $NODE_TAGS}" > /dev/null

# 5. 重启服务
echo "正在重启 sing-box 服务..."
RESTART_RESPONSE=$(curl -s -X POST "$API_BASE/service/restart")

if echo $RESTART_RESPONSE | jq -e '.message' > /dev/null; then
  echo "服务重启成功！"
else
  echo "服务重启失败，正在回滚配置..."
  curl -s -X POST "$API_BASE/config/rollback"
  echo "配置已回滚"
  exit 1
fi

echo "节点更新完成！"
```

---

## 错误处理示例

### 处理批量添加时的部分失败

```bash
RESPONSE=$(curl -s -X POST http://localhost:8080/1.13.0/outbounds/batch \
  -H "Content-Type: application/json" \
  -d '{
    "outbounds": [...]
  }')

# 检查是否有跳过的节点
SKIPPED_COUNT=$(echo $RESPONSE | jq -r '.skipped_count // 0')

if [ "$SKIPPED_COUNT" -gt 0 ]; then
  echo "警告: 有 $SKIPPED_COUNT 个节点已存在，已跳过"
  SKIPPED_TAGS=$(echo $RESPONSE | jq -r '.skipped_tags[]')
  echo "跳过的节点: $SKIPPED_TAGS"
fi

ADDED_COUNT=$(echo $RESPONSE | jq -r '.added_count')
echo "成功添加 $ADDED_COUNT 个新节点"
```

### 处理配置验证失败

```bash
VALIDATE_RESPONSE=$(curl -s -X POST http://localhost:8080/1.13.0/config/validate \
  -H "Content-Type: application/json" \
  -d @new-config.json)

IS_VALID=$(echo $VALIDATE_RESPONSE | jq -r '.valid')

if [ "$IS_VALID" != "true" ]; then
  echo "配置验证失败:"
  echo $VALIDATE_RESPONSE | jq -r '.error'
  exit 1
fi

echo "配置验证通过"
```

---

## Python 客户端示例

```python
import requests
import json

class SingBoxEasyClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = f"{base_url}/1.13.0"

    def parse_nodes(self, subscription):
        """解析订阅节点"""
        response = requests.post(
            f"{self.base_url}/nodes/parse",
            json={"subscription": subscription}
        )
        return response.json()

    def add_outbounds_batch(self, outbounds):
        """批量添加节点"""
        response = requests.post(
            f"{self.base_url}/outbounds/batch",
            json={"outbounds": outbounds}
        )
        return response.json()

    def update_subscription(self, sub_id):
        """更新订阅"""
        response = requests.post(
            f"{self.base_url}/subscriptions/{sub_id}/update"
        )
        return response.json()

    def restart_service(self):
        """重启服务"""
        response = requests.post(f"{self.base_url}/service/restart")
        return response.json()

# 使用示例
client = SingBoxEasyClient()

# 1. 更新订阅获取节点
result = client.update_subscription("sub_1234567890")
nodes = result["nodes"]
print(f"获取到 {len(nodes)} 个节点")

# 2. 批量添加节点
add_result = client.add_outbounds_batch(nodes)
print(f"成功添加 {add_result['added_count']} 个节点")

# 3. 重启服务
restart_result = client.restart_service()
print(restart_result["message"])
```
