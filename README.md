# sing-box-easy

sing-box-easy 是一个带有现代化 Web 界面的 sing-box 配置管理工具。它提供功能完整的仪表盘，用于可视化管理 sing-box 的配置、节点、订阅与服务生命周期。

## ✨ 主要特性

### 🖥️ Web 仪表盘
- **可视化配置管理**：内置 Monaco Editor，支持 JSON 语法高亮、补全与校验。
- **节点与订阅**：直观的节点列表、分组（selector / urltest）编辑、订阅自动更新调度器。
- **初始化向导**：首次启动自动进入 `/init` 引导，分步完成 sing-box 安装、配置生成、Dashboard UI 部署。
- **实时监控**：查看 sing-box 进程状态与日志。
- **现代化 UI**：Vue 3 + TypeScript + TailwindCSS v4 + DaisyUI + PrimeVue。

### ⚙️ 后端核心
- **RESTful API**：覆盖配置、DNS、Inbound/Outbound、路由、订阅、调度器、安装、Dashboard、初始化等共 79 个端点。
- **配置安全**：每次写盘都走 `写临时文件 → sing-box check → 原子替换 → 备份` 流程，可一键 `/config/rollback` 回滚。
- **引用一致性**：删除 outbound 或订阅刷新时自动清理 `selector` / `urltest` 中已失效的引用，避免 sing-box 静默运行时悬挂指针。
- **多协议解析**：Shadowsocks (`ss://`)、VMess (`vmess://`)、Trojan (`trojan://`)。
- **统一响应信封**：所有业务接口返回 `{ code, data, msg }`，HTTP 状态码恒为 200，前端按业务 `code` 分流。
- **订阅自动更新**：cron 调度器（默认每 5 分钟）按订阅配置的 `update_interval` 增量拉取、差分应用。
- **高性能存储**：SQLite + XORM（`modernc.org/sqlite` 纯 Go，无需 CGO）。

## 🚀 快速开始

### 预备条件
- **Go**: 1.25.3+
- **Node.js**: 22.21+（用于构建前端）
- **sing-box**: 需预先安装内核（系统 PATH 中可访问 `sing-box`，或在 `app.yml` 中指定 `binary_path`）

### 🛠️ 编译安装

#### 1. 构建前端
```bash
cd frontend
npm install
npm run build
cd ..
# 构建产物输出到项目根目录的 dist/ 文件夹
```

#### 2. 编译后端
```bash
go build -o sing-box-easy ./main.go
```

### 🏃 运行

1. **准备配置**
   ```bash
   cp app.yml.example app.yml
   # 根据需要编辑 app.yml（端口、sing-box 路径、数据库路径等）
   ```

2. **启动服务**
   ```bash
   ./sing-box-easy
   # 也可显式指定配置文件:
   ./sing-box-easy -c /path/to/app.yml
   ```

3. **访问仪表盘**
   打开浏览器访问 `http://localhost:8080`（默认端口，可通过 `app.yml` 或 `HTTP_PORT` 环境变量覆盖）。
   首次访问会自动进入初始化向导。

## 🐳 Docker 部署

项目提供开箱即用的 Docker 支持，自动处理前后端构建。

```bash
# 1. 准备配置
cp app.yml.example app.yml

# 2. 启动容器
docker-compose up -d
```

更多详情请参考 [Docker 部署指南](DOCKER.md)。

## 📁 目录结构

```
sing-box-easy/
├── app/
│   ├── pkg/                # 后端核心包
│   │   ├── appconfig/      # app.yml 加载
│   │   ├── config/         # sing-box 配置管理 + 校验回滚
│   │   ├── database/       # SQLite + XORM
│   │   ├── service/        # sing-box 进程生命周期
│   │   ├── subscription/   # 订阅 CRUD + cron 自动更新器
│   │   ├── sublink/        # 订阅抓取 + 协议解析
│   │   ├── installer/      # sing-box 与 Dashboard 安装
│   │   └── ...
│   └── routes/v1_12_12/    # 当前版本 API 路由与处理器
├── frontend/               # Vue 3 + Vite + TypeScript 前端
│   └── src/
│       ├── views/          # 路由级页面（含 init-steps/、dashboard/）
│       ├── components/     # 通用组件
│       ├── services/       # axios 客户端，每个 domain 一个模块
│       └── stores/         # Pinia
├── bin/                    # 开发态运行目录（app.yml、config.json、SQLite DB）
├── dist/                   # 前端构建产物（npm run build 生成）
├── doc/                    # 项目文档（API、迁移说明、配置模板）
├── main.go                 # 程序入口
└── README.md
```

## 🛠️ 开发指南

### 后端（dev 模式）
```bash
./dev.sh
# 等价于: DEBUG=true go run . -c bin/app.yml
# 默认监听端口 5100，数据库在 ./bin/sing-box-easy.db
```

### 前端（dev 模式）
```bash
cd frontend
npm run dev
# Vite 默认 http://localhost:5173
# /api/* 请求会被代理到 http://localhost:5100（即后端 dev 端口）
```

> 修改后端 dev 端口时，需要同步改 `bin/app.yml` 的 `server.port` 与 `frontend/vite.config.ts` 中的 proxy target。

### 运行测试
```bash
# 后端
go test ./...

# 单个测试
go test ./app/pkg/subscription -v -run TestApplyChanges
```

## 📝 文档

| 文档 | 内容 |
|------|------|
| [`doc/API_v1.13.0.md`](doc/API_v1.13.0.md) | 完整 v1.12.12 API 接口文档（含通用响应信封、错误码、所有 79 个端点） |
| [`doc/API_Requirements.md`](doc/API_Requirements.md) | 接口总览与初始化向导接口组合说明 |
| [`doc/DATABASE_MIGRATION.md`](doc/DATABASE_MIGRATION.md) | JSON → SQLite 自动迁移指南 |
| [`doc/config.1.12.12.json`](doc/config.1.12.12.json) | sing-box 配置参考模板 |
| [`DOCKER.md`](DOCKER.md) | Docker 部署指南 |
| [`CLAUDE.md`](CLAUDE.md) | 给 Claude Code 等 AI 编辑器的代码协作指南 |

> 文件名 `API_v1.13.0.md` 是历史遗留，其内容当前对应 sing-box `1.12.12`。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License
