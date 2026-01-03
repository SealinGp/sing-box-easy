# sing-box-easy

sing-box-easy 是一个带有现代化 Web 界面的 sing-box 配置管理工具。它提供了一个功能强大的仪表盘，用于可视化管理 sing-box 的配置、节点、订阅和服务状态。

## ✨ 主要特性

### �️ Web 仪表盘
- **可视化配置管理**：内置 Monaco Editor，支持 JSON 语法高亮和自动补全。
- **节点与订阅**：直观的节点列表和订阅管理界面，支持多种协议解析。
- **实时监控**：查看服务运行状态和日志。
- **现代化 UI**：基于 Vue 3 + TailwindCSS 构建，提供流畅的用户体验。

### ⚙️ 后端核心
- **RESTful API**：完整的配置管理和服务控制 API。
- **配置安全**：
  - 自动备份与回滚机制
  -这是配置变更前的预验证
- **节点管理**：支持 Shadowsocks, VMess 等多种协议节点解析。
- **高性能存储**：使用 SQLite + XORM，支持高并发访问。

## 🚀 快速开始

### 预备条件
- **Go**: 1.19+
- **Node.js**: 18+ (用于构建前端)
- **sing-box**: 需要预先安装 sing-box 内核

### 🛠️ 编译安装

这是一个全栈项目，包含前端和后端部分。

#### 1. 构建前端
```bash
cd frontend
npm install
npm run build
cd ..
# 构建产物将自动输出到项目根目录的 dist 文件夹
```

#### 2. 编译后端
```bash
go build -o sing-box-easy ./main.go
```

### 🏃 运行

1. **准备配置**
   ```bash
   cp app.yml.example app.yml
   # 根据需要编辑 app.yml
   ```

2. **启动服务**
   ```bash
   ./sing-box-easy
   ```

3. **访问仪表盘**
   打开浏览器访问：`http://localhost:8080`

## 🐳 Docker 部署

项目提供了开箱即用的 Docker 支持，自动处理前后端构建。

```bash
# 1. 准备配置目录
mkdir -p config
cp app.yml.example app.yml

# 2. 启动容器
docker-compose up -d
```

更多详情请参考 [Docker 部署指南](DOCKER.md)。

## 📁 目录结构

```
sing-box-easy/
├── app/                 # Go 后端核心代码
│   ├── pkg/            # 核心逻辑 (Config, Service, etc.)
│   └── routes/         # API 路由与静态文件服务
├── frontend/           # Vue 3 前端项目
│   ├── src/            # 前端源码
│   └── vite.config.ts  # Vite 配置
├── dist/               # 前端构建产物 (自动生成)
├── doc/                # 文档
├── app.yml             # 应用配置文件
├── main.go             # 程序入口
└── README.md
```

## 🛠️ 开发指南

### 后端开发
```bash
# 运行后端服务 (默认端口 8080)
go run main.go
```

### 前端开发
```bash
cd frontend
npm run dev
# 开发服务器默认运行在 http://localhost:5173
# API 请求会通过代理转发到本地后端 8080 端口
```

## 📝 API 文档

后端提供完整的 RESTful API，详情请查看 [API 文档](doc/API_v1.12.12.md)。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License
