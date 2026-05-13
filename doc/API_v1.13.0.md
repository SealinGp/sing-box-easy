# sing-box-easy API v1.12.12 文档

本文档描述 sing-box-easy 当前实现的 RESTful API 接口（对应 sing-box `1.12.12`）。

> 文件名保留为 `API_v1.13.0.md` 以兼容现有引用，但当前 API 版本为 **v1.12.12**。

---

## 基本信息

- **Base URL**: `/api/1.12.12`
- **Content-Type**: `application/json`
- **HTTP 状态码**: 所有业务接口 **始终返回 `200 OK`**，业务成败由响应体的 `code` 字段表达。
- **传输层错误**（如反序列化失败、Hertz 内部错误）仍可能返回非 200。

---

## 通用响应格式

所有 `/api/1.12.12/*` 接口（除特殊说明外）均使用 `BasicResponse` 信封：

```json
{
  "code": 0,
  "data": { /* 业务数据，错误时为 null */ },
  "msg":  "success"
}
```

### 业务 code 枚举

| Code | 名称 | 说明 |
|------|------|------|
| 0 | `Success` | 成功 |
| 1 | `BadRequest` | 请求参数错误（含反序列化失败、字段缺失等） |
| 2 | `NotFound` | 资源不存在 |
| 3 | `InternalError` | 服务器内部错误 |
| 4 | `ValidationError` | 校验失败（含 sing-box check 失败） |
| 5 | `Conflict` | 资源冲突（如 tag 重复） |
| 6 | `Unauthorized` | 未授权 |
| 7 | `Forbidden` | 操作被禁止 |
| 8 | `ServiceError` | 外部服务错误（sing-box 进程相关） |
| 9 | `ConfigError` | 配置错误 |
| 10 | `OperationFailed` | 操作失败 |

> 错误响应示例：
> ```json
> { "code": 1, "data": null, "msg": "name is required" }
> ```

> 前端应基于 `data.code === 0` 判断成功，而非 HTTP 状态码。

---

## 一、配置文件管理 API

### 1.1 获取当前配置

`GET /api/1.12.12/config`

返回完整的 sing-box 配置（`SingBoxConfig`）。

### 1.2 更新整份配置

`PUT /api/1.12.12/config`

请求体：完整的 sing-box 配置 JSON。写入流程：写临时文件 → `sing-box check` → 备份 → 原子替换。

### 1.3 验证配置

`POST /api/1.12.12/config/validate`

请求体：要验证的完整配置 JSON。仅做校验，不写盘。

响应 `data`：
```json
{ "valid": true, "message": "configuration is valid" }
```

### 1.4 获取备份配置

`GET /api/1.12.12/config/backup`

返回 `config.old.json` 的内容（上一份成功写入的配置）。

### 1.5 回滚配置

`POST /api/1.12.12/config/rollback`

将 `config.old.json` 还原为 `config.json`。

---

## 二、节点解析与 Outbound 管理 API

### 2.1 解析节点链接

`POST /api/1.12.12/nodes/parse`

请求体：
```json
{ "subscription": "ss://...\nvmess://...\ntrojan://..." }
```

响应 `data`：
```json
{
  "message":    "nodes parsed successfully",
  "node_count": 10,
  "nodes":      [ /* option.Outbound 数组 */ ]
}
```

支持协议：`shadowsocks` (`ss://`)、`vmess` (`vmess://`)、`trojan` (`trojan://`)。

### 2.2 获取所有 Outbound

`GET /api/1.12.12/outbounds`

响应 `data`：`{ "outbounds": [...] }`

### 2.3 获取指定 Outbound

`GET /api/1.12.12/outbounds/:tag`

### 2.4 添加单个 Outbound

`POST /api/1.12.12/outbounds`

请求体：完整的 `option.Outbound` 对象（依协议类型字段而异）。

### 2.5 批量添加 Outbound

`POST /api/1.12.12/outbounds/batch`

请求体：
```json
{ "outbounds": [ { /* outbound 1 */ }, { /* outbound 2 */ } ] }
```

响应 `data`：
```json
{
  "message":       "outbounds batch add completed",
  "added_count":   1,
  "added_tags":    ["🇯🇵 日本 02"],
  "skipped_count": 1,
  "skipped_tags":  ["🇯🇵 日本 01"]
}
```

- 已存在的 tag 会进入 `skipped_tags`，不会让整个请求失败。
- 全部都已存在时返回 `OperationFailed`。

### 2.6 更新 Outbound

`PUT /api/1.12.12/outbounds/:tag`

请求体：完整的 outbound 对象。URL 上的 `:tag` 会强制覆盖请求体里的 `tag`（即此接口**不支持改名**）。

### 2.7 删除单个 Outbound

`DELETE /api/1.12.12/outbounds/:tag`

`:tag` 既可以是 tag 字符串，也可以是数字下标。

**自动联动清理**: 删除时会同步将该 tag 从所有 `selector` / `urltest` 类型的 outbound 的 `outbounds` 列表 与 `selector.default` 字段中移除，避免遗留悬挂引用（`sing-box check` 不会检测这种情况）。

### 2.8 批量删除 Outbound

`DELETE /api/1.12.12/outbounds/batch`

请求体：
```json
{ "tags": ["tag1", "tag2"] }
```

响应 `data`：
```json
{
  "message":        "outbounds deleted successfully",
  "deleted_count":  2,
  "deleted_tags":   ["tag1", "tag2"],
  "not_found_tags": []
}
```

不存在的 tag 收集在 `not_found_tags` 中，不会让整个请求失败。**同样会自动清理** selector/urltest 中的引用。

### 2.9 获取所有分组

`GET /api/1.12.12/outbounds/groups`

返回所有 `selector` / `urltest` 类型的 outbound。

### 2.10 更新分组成员

`PUT /api/1.12.12/outbounds/:tag/members`

请求体：
```json
{ "outbounds": ["🇭🇰 香港 01", "🇯🇵 日本 01"] }
```

仅替换分组的 `outbounds` 字段；其他字段不变。

---

## 三、DNS 配置 API

### 3.1 获取 / 更新整份 DNS 配置

- `GET  /api/1.12.12/dns`
- `PUT  /api/1.12.12/dns`

请求体（PUT）：完整的 `option.DNSOptions`。

### 3.2 DNS 服务器

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/dns/servers` | 列表 |
| `POST` | `/api/1.12.12/dns/servers` | 新增 |
| `GET` | `/api/1.12.12/dns/servers/:tag` | 单个查询 |
| `PUT` | `/api/1.12.12/dns/servers/:tag` | 更新 |
| `DELETE` | `/api/1.12.12/dns/servers/:tag` | 删除 |

### 3.3 Hosts

- `GET /api/1.12.12/dns/hosts`
- `PUT /api/1.12.12/dns/hosts`

请求体（PUT）：
```json
{ "hosts": { "home.example.com": ["192.168.1.1"] } }
```

### 3.4 DNS 规则

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/dns/rules` | 列表 |
| `POST` | `/api/1.12.12/dns/rules` | 新增 |
| `PUT` | `/api/1.12.12/dns/rules/:index` | 按下标更新 |
| `DELETE` | `/api/1.12.12/dns/rules/:index` | 按下标删除 |

`:index` 从 0 开始。

---

## 四、Inbound 配置 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/inbounds` | 列表 |
| `POST` | `/api/1.12.12/inbounds` | 新增 |
| `GET` | `/api/1.12.12/inbounds/:tag` | 单个查询 |
| `PUT` | `/api/1.12.12/inbounds/:tag` | 更新 |
| `DELETE` | `/api/1.12.12/inbounds/:tag` | 删除 |

---

## 五、Route 路由配置 API

### 5.1 路由规则

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/route/rules` | 列表 |
| `POST` | `/api/1.12.12/route/rules` | 新增 |
| `PUT` | `/api/1.12.12/route/rules/:index` | 更新 |
| `DELETE` | `/api/1.12.12/route/rules/:index` | 删除 |

### 5.2 规则集

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/route/rule-sets` | 列表 |
| `POST` | `/api/1.12.12/route/rule-sets` | 新增 |
| `GET` | `/api/1.12.12/route/rule-sets/:tag` | 单个查询 |
| `PUT` | `/api/1.12.12/route/rule-sets/:tag` | 更新 |
| `DELETE` | `/api/1.12.12/route/rule-sets/:tag` | 删除 |

### 5.3 兜底策略

- `GET /api/1.12.12/route/final` → `{ "final": "🐠 兜底策略" }`
- `PUT /api/1.12.12/route/final` ← `{ "final": "➡️ 直连" }`

---

## 六、日志配置 API

- `GET /api/1.12.12/log`
- `PUT /api/1.12.12/log`

请求体（PUT）：
```json
{ "disabled": false, "level": "info", "timestamp": true }
```

---

## 七、实验性功能 API

### 7.1 Clash API

- `GET /api/1.12.12/experimental/clash-api`
- `PUT /api/1.12.12/experimental/clash-api`

### 7.2 Cache File

- `GET /api/1.12.12/experimental/cache-file`
- `PUT /api/1.12.12/experimental/cache-file`

### 7.3 V2Ray API

- `GET /api/1.12.12/experimental/v2ray-api`
- `PUT /api/1.12.12/experimental/v2ray-api`

---

## 八、服务控制 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/service/status` | 通过 `pgrep` 检测进程状态 |
| `POST` | `/api/1.12.12/service/start` | 启动前先 `sing-box check` |
| `POST` | `/api/1.12.12/service/stop` | SIGTERM → 必要时 SIGKILL |
| `POST` | `/api/1.12.12/service/restart` | 启动前先校验配置 |

`/status` 响应 `data`：
```json
{ "status": "running", "running": true }
```

---

## 九、订阅管理 API

### 9.1 CRUD

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/subscriptions` | 列表 |
| `POST` | `/api/1.12.12/subscriptions` | 添加 |
| `GET` | `/api/1.12.12/subscriptions/:id` | 单个查询 |
| `PUT` | `/api/1.12.12/subscriptions/:id` | 更新（仅元数据，如 `name`/`url`/`auto_update`） |
| `DELETE` | `/api/1.12.12/subscriptions/:id` | 删除 |

POST 请求体：
```json
{
  "name":            "我的订阅",
  "url":             "https://example.com/sub",
  "enabled":         true,
  "auto_update":     true,
  "update_interval": "24h"
}
```

### 9.2 手动触发订阅内容更新

`POST /api/1.12.12/subscriptions/:id/update`

响应 `data`（结构与 cron 自动更新一致）：
```json
{
  "message":      "subscription updated successfully",
  "id":           "sub_1234567890",
  "added_tags":   ["🇭🇰 香港 03 hk3.example.com:8388"],
  "updated_tags": ["🇯🇵 日本 01 jp1.example.com:8388"],
  "deleted_keys": ["tw.example.com:443"],
  "added":        1,
  "updated":      1,
  "deleted":      1
}
```

- `added_tags` / `updated_tags`：经 `tag + server:port` 去重后的最终 tag。
- `deleted_keys`：被删除的 `server:port` 标识（不是 tag）。
- **同步清理**: 被删除的 tag 与被改名的 tag 会自动从所有 `selector` / `urltest` 的 `outbounds` 列表和 `selector.default` 中清理 / 重写。

---

## 十、调度器（订阅自动更新）API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/scheduler/status` | 是否运行中、上次检查时间 |
| `POST` | `/api/1.12.12/scheduler/start` | 启动 cron 调度器 |
| `POST` | `/api/1.12.12/scheduler/stop` | 停止 |
| `POST` | `/api/1.12.12/scheduler/trigger` | 立即触发一次全量检查 |
| `GET` | `/api/1.12.12/scheduler/jobs` | 各订阅的统计信息（成功/失败次数、最后错误） |

---

## 十一、sing-box 安装与升级 API

### 11.1 安装

`POST /api/1.12.12/install`

请求体：
```json
{ "version": "1.12.12", "beta": false }
```

响应 `data`：
```json
{ "message": "sing-box installation started", "task_id": "install_xxx" }
```

### 11.2 升级

`POST /api/1.12.12/update`

请求体同上，但通常 `version` 指向新版本。

### 11.3 任务状态查询

`GET /api/1.12.12/install/task/:task_id`

返回任务进度（`pending` / `running` / `success` / `failed`）与日志。

### 11.4 安装状态

`GET /api/1.12.12/install/status`

```json
{
  "installing": false,
  "installed":  true,
  "version":    "1.12.12",
  "message":    "sing-box is installed"
}
```

---

## 十二、Dashboard UI 管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/1.12.12/dashboard/download` | 下载并安装 dashboard（异步任务） |
| `POST` | `/api/1.12.12/dashboard/upload` | 上传自定义 dashboard zip |
| `GET` | `/api/1.12.12/dashboard/task/:task_id` | 任务进度 |
| `GET` | `/api/1.12.12/dashboard/status` | 当前 dashboard 安装信息 |

---

## 十三、初始化向导 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/1.12.12/init/status` | 初始化状态（含分步状态） |
| `POST` | `/api/1.12.12/init/complete` | 标记初始化完成 |
| `POST` | `/api/1.12.12/init/reset` | 重置初始化状态（重新进入向导） |

---

## 十四、模板 API

`GET /api/1.12.12/templates/rule-sets`

返回内置的默认 rule-set 集合（用于初始化向导一键导入）。

---

## 配置修改安全机制

所有写盘的接口（POST/PUT/DELETE）都经过同一条 `Manager.UpdateConfig` 路径：

1. `GetConfig()` 重新从磁盘读取最新 `config.json`
2. 执行用户提供的更新函数
3. 写入 `config_new.json`
4. 调用 `sing-box check -c config_new.json`
5. 校验成功：`config.json` → `config.old.json`（备份），`config_new.json` → `config.json`（原子替换）
6. 校验失败：保留原 `config.json`，删除 `config_new.json`，返回 `ValidationError`

可随时通过 `POST /api/1.12.12/config/rollback` 把 `config.old.json` 还原回去。

---

## 已知约束

- `sing-box check` 只校验协议层字段，不校验 `selector` / `urltest` 的 `outbounds` 列表里是否还有失效 tag。删除 / 订阅更新接口已经在应用层做了清理（见 `app/pkg/config/group_refs.go`），手工 `PUT /config` 时仍需调用方保证一致性。
- `Manager.UpdateConfig` 当前没有进程内锁，并发写存在 TOCTOU 窗口。生产部署下应保证仅一个写入入口（前端 + cron 调度器是串行触发）。
- HTTP 状态码统一为 200。前端务必按 `data.code` 分流。
