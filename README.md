# sing-box-easy

sing-box-easy 是一个 sing-box 配置管理和服务控制的 RESTful API 服务。

## 功能特性

- 🔧 完整的 sing-box 配置管理（Inbound、Outbound、DNS、Route 等）
- 🚀 服务生命周期控制（启动、停止、重启）
- 📦 节点订阅管理与自动更新
- 🔐 配置安全机制（自动备份、验证、回滚）
- 📝 支持 shadowsocks、vmess 等多种协议节点解析
- 🌐 RESTful API 设计，易于集成

## 快速开始

### 1. 安装

```bash
# 克隆项目
git clone https://github.com/SealinGp/sing-box-easy.git
cd sing-box-easy

# 编译
go build -o sing-box-easy ./main.go
```

### 2. 配置

复制配置文件示例并修改：

```bash
cp app.yml.example app.yml
```

编辑 `app.yml` 配置您的环境：

```yaml
server:
  port: "8080"

sing_box:
  config_path: "/etc/sing-box/config.json"
  binary_path: "sing-box"
  subscription_path: "/etc/sing-box/subscriptions.json"
```

详细配置说明请查看 [配置文档](doc/Configuration.md)。

### 3. 运行

```bash
# 使用默认配置文件 (app.yml)
./sing-box-easy

# 指定配置文件
./sing-box-easy -c /path/to/config.yml

# 使用环境变量覆盖端口
HTTP_PORT=9090 ./sing-box-easy
```

## API 文档

完整的 API 文档请查看 [API v1.13.0 文档](doc/API_v1.13.0.md)。

### 主要 API 端点

#### 配置管理
- `GET /1.13.0/config` - 获取当前配置
- `POST /1.13.0/config/validate` - 验证配置
- `POST /1.13.0/config/rollback` - 回滚配置

#### 节点管理
- `GET /1.13.0/outbounds` - 获取所有节点
- `POST /1.13.0/outbounds` - 添加节点
- `PUT /1.13.0/outbounds/:tag` - 更新节点
- `DELETE /1.13.0/outbounds/:tag` - 删除节点

#### 服务控制
- `GET /1.13.0/service/status` - 获取服务状态
- `POST /1.13.0/service/start` - 启动服务
- `POST /1.13.0/service/stop` - 停止服务
- `POST /1.13.0/service/restart` - 重启服务

#### 订阅管理
- `GET /1.13.0/subscriptions` - 获取所有订阅
- `POST /1.13.0/subscriptions` - 添加订阅
- `POST /1.13.0/subscriptions/:id/update` - 更新订阅内容

更多接口请参考完整的 [API 文档](doc/API_v1.13.0.md)。

## 使用示例

### 解析节点

```bash
curl -X POST http://localhost:8080/1.13.0/nodes/parse \
  -H "Content-Type: application/json" \
  -d '{
    "subscription": "ss://YWVzLTEyOC1nY206dGVzdA==@192.168.1.1:8888"
  }'
```

### 获取服务状态

```bash
curl http://localhost:8080/1.13.0/service/status
```

### 添加订阅

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

## 配置安全机制

所有配置修改操作都遵循安全流程：

1. 创建临时配置 `config_new.json`
2. 使用 `sing-box check` 验证配置
3. 验证通过后：
   - 备份当前配置到 `config.old.json`
   - 应用新配置到 `config.json`
4. 验证失败时保持原配置不变

任何时候都可以通过 `/1.13.0/config/rollback` 回滚到上一个配置。

## 目录结构

```
sing-box-easy/
├── app/
│   ├── pkg/
│   │   ├── appconfig/      # 应用配置
│   │   ├── config/          # sing-box 配置管理
│   │   ├── service/         # 服务控制
│   │   ├── subscription/    # 订阅管理
│   │   └── sublink/         # 节点解析
│   ├── routes/
│   │   └── v1_13_0/         # v1.13.0 API handlers
│   └── svr.go
├── doc/
│   ├── API_v1.13.0.md       # API 文档
│   ├── Configuration.md     # 配置文档
│   └── Features.md          # 功能需求文档
├── app.yml                  # 配置文件
├── app.yml.example          # 配置文件示例
├── main.go                  # 程序入口
└── README.md
```

## 依赖

- Go 1.19+
- [Hertz](https://github.com/cloudwego/hertz) - HTTP 框架
- [sing-box](https://github.com/SagerNet/sing-box) - 代理工具（需要安装）

## 开发

### 编译

```bash
go build -o sing-box-easy ./main.go
```

### 运行测试

```bash
go test ./...
```

## 部署

### Systemd 服务

创建 `/etc/systemd/system/sing-box-easy.service`:

```ini
[Unit]
Description=sing-box-easy API Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/sing-box-easy -c /etc/sing-box-easy/config.yml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable sing-box-easy
sudo systemctl start sing-box-easy
```

### Docker 部署

```dockerfile
FROM golang:1.19 AS builder
WORKDIR /app
COPY . .
RUN go build -o sing-box-easy ./main.go

FROM debian:bullseye-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/sing-box-easy /usr/local/bin/
COPY app.yml /etc/sing-box-easy/config.yml
EXPOSE 8080
CMD ["sing-box-easy", "-c", "/etc/sing-box-easy/config.yml"]
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 相关链接

- [sing-box 官方文档](https://sing-box.sagernet.org/)
- [API 文档](doc/API_v1.13.0.md)
- [配置文档](doc/Configuration.md)
