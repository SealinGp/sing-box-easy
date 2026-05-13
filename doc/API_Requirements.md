# sing-box-easy 前端所需接口整理

> **状态**：本文档当初作为前后端联调时的接口规划文档。第一节列出当时已存在的接口；
> 第二节当时是"待新增"，目前**均已实现**。第三节描述初始化向导的接口组合，仍然适用。
> 完整接口定义见 [`API_v1.13.0.md`](./API_v1.13.0.md)。
>
> 所有路径前缀为 `/api/1.12.12`（原文档遗漏了 `/api`，已修正）。

---

## 一、当前实现的接口（完整目录）

### 1. 配置管理
- ✅ `GET    /api/1.12.12/config`              — 获取当前配置
- ✅ `PUT    /api/1.12.12/config`              — 整份替换配置
- ✅ `POST   /api/1.12.12/config/validate`     — 校验配置
- ✅ `GET    /api/1.12.12/config/backup`       — 获取备份配置
- ✅ `POST   /api/1.12.12/config/rollback`     — 回滚至备份

### 2. Outbound 节点管理
- ✅ `GET    /api/1.12.12/outbounds`           — 列表
- ✅ `POST   /api/1.12.12/outbounds`           — 新增单个
- ✅ `POST   /api/1.12.12/outbounds/batch`     — 批量新增
- ✅ `DELETE /api/1.12.12/outbounds/batch`     — **批量删除（新增）**
- ✅ `GET    /api/1.12.12/outbounds/groups`    — 仅返回 selector/urltest 分组
- ✅ `GET    /api/1.12.12/outbounds/:tag`      — 单个查询
- ✅ `PUT    /api/1.12.12/outbounds/:tag`      — 更新
- ✅ `DELETE /api/1.12.12/outbounds/:tag`      — 删除（**自动清理 selector/urltest 引用**）
- ✅ `PUT    /api/1.12.12/outbounds/:tag/members` — 更新分组成员

### 3. DNS 配置
- ✅ `GET    /api/1.12.12/dns`                 — **整份 DNS 配置（新增）**
- ✅ `PUT    /api/1.12.12/dns`                 — **整份 DNS 配置写入（新增）**
- ✅ `GET    /api/1.12.12/dns/servers`         — DNS 服务器列表
- ✅ `POST   /api/1.12.12/dns/servers`         — 新增
- ✅ `GET    /api/1.12.12/dns/servers/:tag`    — 单个查询
- ✅ `PUT    /api/1.12.12/dns/servers/:tag`    — 更新
- ✅ `DELETE /api/1.12.12/dns/servers/:tag`    — 删除
- ✅ `GET    /api/1.12.12/dns/hosts`           — hosts
- ✅ `PUT    /api/1.12.12/dns/hosts`           — hosts 写入
- ✅ `GET    /api/1.12.12/dns/rules`           — DNS 规则列表
- ✅ `POST   /api/1.12.12/dns/rules`           — 新增
- ✅ `PUT    /api/1.12.12/dns/rules/:index`    — 按下标更新
- ✅ `DELETE /api/1.12.12/dns/rules/:index`    — 按下标删除

### 4. Inbound 配置
- ✅ `GET    /api/1.12.12/inbounds`            — 列表
- ✅ `POST   /api/1.12.12/inbounds`            — 新增
- ✅ `GET    /api/1.12.12/inbounds/:tag`       — 单个查询
- ✅ `PUT    /api/1.12.12/inbounds/:tag`       — 更新
- ✅ `DELETE /api/1.12.12/inbounds/:tag`       — 删除

### 5. Route 配置
- ✅ `GET    /api/1.12.12/route/rules`
- ✅ `POST   /api/1.12.12/route/rules`
- ✅ `PUT    /api/1.12.12/route/rules/:index`
- ✅ `DELETE /api/1.12.12/route/rules/:index`
- ✅ `GET    /api/1.12.12/route/rule-sets`
- ✅ `POST   /api/1.12.12/route/rule-sets`
- ✅ `GET    /api/1.12.12/route/rule-sets/:tag`
- ✅ `PUT    /api/1.12.12/route/rule-sets/:tag`
- ✅ `DELETE /api/1.12.12/route/rule-sets/:tag`
- ✅ `GET    /api/1.12.12/route/final`
- ✅ `PUT    /api/1.12.12/route/final`

### 6. 日志和实验性配置
- ✅ `GET    /api/1.12.12/log`
- ✅ `PUT    /api/1.12.12/log`
- ✅ `GET    /api/1.12.12/experimental/clash-api`
- ✅ `PUT    /api/1.12.12/experimental/clash-api`
- ✅ `GET    /api/1.12.12/experimental/cache-file`
- ✅ `PUT    /api/1.12.12/experimental/cache-file`
- ✅ `GET    /api/1.12.12/experimental/v2ray-api`  — **新增**
- ✅ `PUT    /api/1.12.12/experimental/v2ray-api`  — **新增**

### 7. 服务控制
- ✅ `GET    /api/1.12.12/service/status`
- ✅ `POST   /api/1.12.12/service/start`
- ✅ `POST   /api/1.12.12/service/stop`
- ✅ `POST   /api/1.12.12/service/restart`

### 8. 订阅管理
- ✅ `GET    /api/1.12.12/subscriptions`
- ✅ `POST   /api/1.12.12/subscriptions`
- ✅ `GET    /api/1.12.12/subscriptions/:id`        — **单个查询（新增）**
- ✅ `PUT    /api/1.12.12/subscriptions/:id`
- ✅ `DELETE /api/1.12.12/subscriptions/:id`
- ✅ `POST   /api/1.12.12/subscriptions/:id/update` — 手动触发；**自动清理 selector/urltest 引用**

### 9. 订阅自动更新调度器
- ✅ `GET    /api/1.12.12/scheduler/status`
- ✅ `POST   /api/1.12.12/scheduler/start`
- ✅ `POST   /api/1.12.12/scheduler/stop`
- ✅ `POST   /api/1.12.12/scheduler/trigger`
- ✅ `GET    /api/1.12.12/scheduler/jobs`

### 10. 节点解析
- ✅ `POST   /api/1.12.12/nodes/parse`

### 11. sing-box 安装管理
- ✅ `POST   /api/1.12.12/install`
- ✅ `GET    /api/1.12.12/install/task/:task_id`   — 任务进度查询
- ✅ `GET    /api/1.12.12/install/status`
- ✅ `POST   /api/1.12.12/update`

### 12. Dashboard UI 管理
- ✅ `POST   /api/1.12.12/dashboard/download`
- ✅ `POST   /api/1.12.12/dashboard/upload`        — 自定义上传
- ✅ `GET    /api/1.12.12/dashboard/task/:task_id`
- ✅ `GET    /api/1.12.12/dashboard/status`

### 13. 初始化向导
- ✅ `GET    /api/1.12.12/init/status`
- ✅ `POST   /api/1.12.12/init/complete`
- ✅ `POST   /api/1.12.12/init/reset`

### 14. 配置模板
- ✅ `GET    /api/1.12.12/templates/rule-sets`

---

## 二、初始化向导接口组合

#### 步骤 1: 安装 sing-box
```
POST /api/1.12.12/install
GET  /api/1.12.12/install/task/:task_id   (轮询直到 success)
```

#### 步骤 2: 下载 dashboard（如果需要）
```
POST /api/1.12.12/dashboard/download
GET  /api/1.12.12/dashboard/task/:task_id (轮询直到 success)
```

#### 步骤 3: 配置生成（分模块配置）

**3.1 日志配置**
```
PUT /api/1.12.12/log
```

**3.2 实验性配置**
```
PUT /api/1.12.12/experimental/clash-api
PUT /api/1.12.12/experimental/cache-file
PUT /api/1.12.12/experimental/v2ray-api   (可选)
```

**3.3 Inbound 配置**
```
POST /api/1.12.12/inbounds  (tun-in)
POST /api/1.12.12/inbounds  (mixed-in)
```

**3.4 Outbound 配置**
```
# 添加基础节点（直连、block 等）
POST /api/1.12.12/outbounds

# 通过订阅导入节点
POST /api/1.12.12/subscriptions          (创建订阅)
POST /api/1.12.12/subscriptions/:id/update   (手动拉取一次)

# 或：解析单条节点链接后批量加入
POST /api/1.12.12/nodes/parse
POST /api/1.12.12/outbounds/batch

# 创建分组节点（selector / urltest）
POST /api/1.12.12/outbounds
```

**3.5 规则集配置**
```
GET  /api/1.12.12/templates/rule-sets    (获取默认模板)
POST /api/1.12.12/route/rule-sets         (按模板逐项添加)
```

**3.6 DNS 配置**
```
POST /api/1.12.12/dns/servers   (dns_direct / dns_proxy / 可选 dns_lan)
POST /api/1.12.12/dns/rules     (多次)
```

**3.7 路由配置**
```
POST /api/1.12.12/route/rules
PUT  /api/1.12.12/route/final
```

#### 步骤 4: 完成初始化
```
POST /api/1.12.12/init/complete
```

#### 步骤 5: 启动 sing-box 服务
```
POST /api/1.12.12/service/start
```

---

## 三、设计说明与历史决策

### 3.1 异步任务处理
安装、下载等耗时操作使用 task_id 异步模式：

1. POST 启动操作返回 `task_id`
2. 前端轮询 `GET .../task/:task_id` 直到任务 `status = success` 或 `failed`
3. 任务进度与错误信息由 task 接口返回

### 3.2 初始化状态持久化
初始化状态存储于 SQLite（`init_state` 表），含分步标志位（如 `sing_box_installed`、`config_generated`、`dashboard_installed`）和完成时间戳。

### 3.3 默认规则集
内置在代码层，通过 `GET /templates/rule-sets` 暴露给前端，避免硬编码。

### 3.4 响应信封
所有 `/api/1.12.12/*` 接口统一使用 `BasicResponse{code, data, msg}` 信封，HTTP 状态码恒为 200。详见 [`API_v1.13.0.md`](./API_v1.13.0.md#通用响应格式)。

### 3.5 组引用一致性
删除 outbound（单个 / 批量）以及订阅更新会自动从所有 `selector` / `urltest` 的 `outbounds` 列表和 `selector.default` 字段中清理对应 tag，避免 `sing-box check` 通过但运行时分组指向失效节点的情况。实现见 `app/pkg/config/group_refs.go::PruneGroupReferences`。
