# sing-box-easy 前端所需接口整理

## 一、已有接口

### 1. 配置管理
- ✅ GET /1.12.12/config - 获取当前配置
- ✅ POST /1.12.12/config/validate - 验证配置
- ✅ GET /1.12.12/config/backup - 获取备份配置
- ✅ POST /1.12.12/config/rollback - 回滚配置

### 2. 节点管理
- ✅ GET /1.12.12/outbounds - 获取所有节点
- ✅ POST /1.12.12/outbounds - 添加单个节点
- ✅ POST /1.12.12/outbounds/batch - 批量添加节点
- ✅ GET /1.12.12/outbounds/:tag - 获取指定节点
- ✅ PUT /1.12.12/outbounds/:tag - 更新节点
- ✅ DELETE /1.12.12/outbounds/:tag - 删除节点
- ✅ GET /1.12.12/outbounds/groups - 获取分组节点
- ✅ PUT /1.12.12/outbounds/:tag/members - 更新分组成员

### 3. DNS 配置
- ✅ GET /1.12.12/dns/servers - 获取所有 DNS 服务器
- ✅ POST /1.12.12/dns/servers - 添加 DNS 服务器
- ✅ PUT /1.12.12/dns/servers/:tag - 更新 DNS 服务器
- ✅ DELETE /1.12.12/dns/servers/:tag - 删除 DNS 服务器
- ✅ GET /1.12.12/dns/hosts - 获取 hosts
- ✅ PUT /1.12.12/dns/hosts - 更新 hosts
- ✅ GET /1.12.12/dns/rules - 获取 DNS 规则
- ✅ POST /1.12.12/dns/rules - 添加 DNS 规则
- ✅ PUT /1.12.12/dns/rules/:index - 更新 DNS 规则
- ✅ DELETE /1.12.12/dns/rules/:index - 删除 DNS 规则

### 4. Inbound 配置
- ✅ GET /1.12.12/inbounds - 获取所有 inbound
- ✅ POST /1.12.12/inbounds - 添加 inbound
- ✅ PUT /1.12.12/inbounds/:tag - 更新 inbound
- ✅ DELETE /1.12.12/inbounds/:tag - 删除 inbound

### 5. Route 配置
- ✅ GET /1.12.12/route/rules - 获取路由规则
- ✅ POST /1.12.12/route/rules - 添加路由规则
- ✅ PUT /1.12.12/route/rules/:index - 更新路由规则
- ✅ DELETE /1.12.12/route/rules/:index - 删除路由规则
- ✅ GET /1.12.12/route/rule-sets - 获取规则集
- ✅ POST /1.12.12/route/rule-sets - 添加规则集
- ✅ PUT /1.12.12/route/rule-sets/:tag - 更新规则集
- ✅ DELETE /1.12.12/route/rule-sets/:tag - 删除规则集
- ✅ GET /1.12.12/route/final - 获取兜底策略
- ✅ PUT /1.12.12/route/final - 更新兜底策略

### 6. 日志和实验性配置
- ✅ GET /1.12.12/log - 获取日志配置
- ✅ PUT /1.12.12/log - 更新日志配置
- ✅ GET /1.12.12/experimental/clash-api - 获取 Clash API 配置
- ✅ PUT /1.12.12/experimental/clash-api - 更新 Clash API 配置
- ✅ GET /1.12.12/experimental/cache-file - 获取缓存配置
- ✅ PUT /1.12.12/experimental/cache-file - 更新缓存配置

### 7. 服务控制
- ✅ GET /1.12.12/service/status - 获取服务状态
- ✅ POST /1.12.12/service/start - 启动服务
- ✅ POST /1.12.12/service/stop - 停止服务
- ✅ POST /1.12.12/service/restart - 重启服务

### 8. 订阅管理
- ✅ GET /1.12.12/subscriptions - 获取所有订阅
- ✅ POST /1.12.12/subscriptions - 添加订阅
- ✅ PUT /1.12.12/subscriptions/:id - 更新订阅
- ✅ DELETE /1.12.12/subscriptions/:id - 删除订阅
- ✅ POST /1.12.12/subscriptions/:id/update - 更新订阅内容

### 9. 节点解析
- ✅ POST /1.12.12/nodes/parse - 解析节点

---

## 二、需要新增的接口

### 1. sing-box 安装管理

#### 1.1 安装 sing-box
**接口**: `POST /1.12.12/install`

**作用**: 安装 sing-box（支持指定版本）

**请求体**:
```json
{
  "version": "1.12.12",  // 可选，不指定则安装最新版
  "beta": false         // 可选，是否安装 beta 版本
}
```

**响应**:
```json
{
  "message": "sing-box installation started",
  "task_id": "install_xxx"
}
```

#### 1.2 获取安装状态
**接口**: `GET /1.12.12/install/status`

**响应**:
```json
{
  "installing": false,
  "installed": true,
  "version": "1.12.12",
  "message": "sing-box is installed"
}
```

#### 1.3 更新 sing-box
**接口**: `POST /1.12.12/update`

**请求体**:
```json
{
  "version": "1.14.0",  // 可选
  "beta": false
}
```

---

### 2. Dashboard UI 管理

#### 2.1 下载 zashboard
**接口**: `POST /1.12.12/dashboard/download`

**作用**: 下载并安装 zashboard UI 到指定目录

**请求体**:
```json
{
  "target_dir": "/etc/sing-box/ui"  // 可选，默认从配置读取
}
```

**响应**:
```json
{
  "message": "dashboard download started",
  "task_id": "download_xxx"
}
```

#### 2.2 获取 dashboard 状态
**接口**: `GET /1.12.12/dashboard/status`

**响应**:
```json
{
  "downloading": false,
  "installed": true,
  "path": "/etc/sing-box/ui"
}
```

---

### 3. 初始化管理

#### 3.1 获取初始化状态
**接口**: `GET /1.12.12/init/status`

**响应**:
```json
{
  "initialized": false,
  "steps": {
    "sing_box_installed": false,
    "config_generated": false,
    "dashboard_installed": false
  }
}
```

#### 3.2 完成初始化
**接口**: `POST /1.12.12/init/complete`

**作用**: 标记初始化已完成

---

### 4. 配置模板

#### 4.1 获取默认规则集列表
**接口**: `GET /1.12.12/templates/rule-sets`

**作用**: 获取推荐的规则集列表（用于初始化）

**响应**:
```json
{
  "rule_sets": [
    {
      "tag": "geosite-cn",
      "type": "remote",
      "format": "binary",
      "url": "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/cn.srs",
      "download_detour": "➡️ 直连"
    },
    ...
  ]
}
```

---

### 5. 静态文件服务（SSR）

#### 5.1 服务前端静态文件
**路由**: `GET /*` (除了 API 路由外的所有路由)

**作用**: 服务前端构建后的静态文件，支持 SPA 路由

---

## 三、可以联合使用的接口组合

### 3.1 初始化流程

#### 步骤 1: 安装 sing-box
```
POST /1.12.12/install
```

#### 步骤 2: 下载 dashboard（如果需要）
```
POST /1.12.12/dashboard/download
```

#### 步骤 3: 配置生成（分模块配置）

**3.1 日志配置**:
```
PUT /1.12.12/log
```

**3.2 Clash API 配置**:
```
PUT /1.12.12/experimental/clash-api
PUT /1.12.12/experimental/cache-file
```

**3.3 Inbound 配置**:
```
POST /1.12.12/inbounds (tun-in)
POST /1.12.12/inbounds (mixed-in)
```

**3.4 Outbound 配置**:
```
# 添加基础节点（直连、block）
POST /1.12.12/outbounds

# 解析订阅/节点
POST /1.12.12/nodes/parse
或
POST /1.12.12/subscriptions + POST /1.12.12/subscriptions/:id/update

# 批量添加节点
POST /1.12.12/outbounds/batch

# 创建分组节点
POST /1.12.12/outbounds (selector/urltest)
```

**3.5 规则集配置**:
```
# 获取默认规则集模板
GET /1.12.12/templates/rule-sets

# 批量添加规则集
POST /1.12.12/route/rule-sets (多次调用)
```

**3.6 DNS 配置**:
```
# 添加 DNS 服务器
POST /1.12.12/dns/servers (dns_direct)
POST /1.12.12/dns/servers (dns_proxy)
POST /1.12.12/dns/servers (dns_lan, 可选)

# 添加 DNS 规则
POST /1.12.12/dns/rules (多次调用)
```

**3.7 路由配置**:
```
# 添加路由规则
POST /1.12.12/route/rules (多次调用)

# 设置兜底策略
PUT /1.12.12/route/final
```

#### 步骤 4: 完成初始化
```
POST /1.12.12/init/complete
```

---

## 四、接口优先级

### 高优先级（初始化必需）
1. ✅ POST /1.12.12/install
2. ✅ GET /1.12.12/install/status
3. ✅ POST /1.12.12/dashboard/download
4. ✅ GET /1.12.12/dashboard/status
5. ✅ GET /1.12.12/init/status
6. ✅ POST /1.12.12/init/complete
7. ✅ GET /1.12.12/templates/rule-sets

### 中优先级（增强功能）
1. ✅ POST /1.12.12/update
2. ✅ 静态文件服务

---

## 五、实现建议

### 5.1 异步任务处理

对于耗时操作（安装、下载），建议使用异步任务：

1. 创建任务管理器
2. 返回 task_id
3. 前端轮询状态接口

### 5.2 初始化状态持久化

使用配置文件或数据库存储初始化状态：
```json
{
  "initialized": true,
  "init_time": "2025-01-01T00:00:00Z",
  "sing_box_version": "1.12.12"
}
```

### 5.3 默认规则集配置

将默认规则集配置写入配置文件或代码中，便于前端获取。
