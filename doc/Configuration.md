# sing-box-easy 配置说明

## 配置文件

sing-box-easy 使用 YAML 格式的配置文件来管理应用程序设置。

### 默认配置文件路径

默认配置文件为项目根目录下的 `app.yml`。

### 使用自定义配置文件

通过命令行参数 `-c` 指定配置文件路径：

```bash
./sing-box-easy -c /path/to/your/config.yml
```

## 配置项说明

### 完整配置示例

```yaml
# sing-box-easy 应用配置文件

server:
  # HTTP 服务器端口
  port: "8080"

sing_box:
  # sing-box 配置文件路径
  config_path: "/etc/sing-box/config.json"

  # sing-box 可执行文件路径（留空则使用 PATH 中的 sing-box）
  binary_path: "sing-box"

  # 订阅配置文件路径
  subscription_path: "/etc/sing-box/subscriptions.json"
```

### 配置项详解

#### server 部分

- `port`: HTTP 服务器监听端口
  - 类型: 字符串
  - 默认值: `"8080"`
  - 说明: API 服务监听的端口号
  - 注意: 可以通过环境变量 `HTTP_PORT` 覆盖此配置

#### sing_box 部分

- `config_path`: sing-box 配置文件路径
  - 类型: 字符串
  - 默认值: `"/etc/sing-box/config.json"`
  - 说明: sing-box 的主配置文件路径，所有配置管理 API 都会操作此文件

- `binary_path`: sing-box 可执行文件路径
  - 类型: 字符串
  - 默认值: `"sing-box"`
  - 说明: sing-box 二进制文件的路径
  - 留空或设置为 `"sing-box"` 将使用系统 PATH 中的 sing-box

- `subscription_path`: 订阅配置文件路径
  - 类型: 字符串
  - 默认值: `"/etc/sing-box/subscriptions.json"`
  - 说明: 存储订阅信息的 JSON 文件路径

## 启动方式

### 1. 使用默认配置

```bash
./sing-box-easy
```

这将使用项目根目录下的 `app.yml` 作为配置文件。

### 2. 指定配置文件

```bash
./sing-box-easy -c /etc/sing-box-easy/config.yml
```

### 3. 使用环境变量覆盖端口

```bash
HTTP_PORT=9090 ./sing-box-easy -c config.yml
```

环境变量 `HTTP_PORT` 的优先级高于配置文件中的 `server.port`。

## 配置文件位置建议

### 开发环境

将 `app.yml` 放在项目根目录：

```
sing-box-easy/
├── app.yml
├── main.go
└── ...
```

### 生产环境

建议将配置文件放在系统配置目录：

```bash
# Linux
/etc/sing-box-easy/config.yml

# 启动命令
/usr/local/bin/sing-box-easy -c /etc/sing-box-easy/config.yml
```

## 配置验证

应用启动时会验证配置文件：

- 如果配置文件不存在或格式错误，程序会输出错误信息并退出
- 未指定的配置项将使用默认值

## 示例配置

### 最小配置

```yaml
server:
  port: "8080"
```

其他配置项将使用默认值。

### Docker 部署配置

```yaml
server:
  port: "8080"

sing_box:
  config_path: "/data/sing-box/config.json"
  binary_path: "/usr/local/bin/sing-box"
  subscription_path: "/data/sing-box/subscriptions.json"
```

### 本地开发配置

```yaml
server:
  port: "3000"

sing_box:
  config_path: "./config/sing-box-config.json"
  binary_path: "sing-box"
  subscription_path: "./config/subscriptions.json"
```

## 注意事项

1. **配置文件权限**: 确保应用程序有读取配置文件的权限
2. **路径配置**: 使用绝对路径可以避免工作目录变化导致的问题
3. **端口冲突**: 确保配置的端口未被其他程序占用
4. **sing-box 路径**: 如果 sing-box 不在 PATH 中，需要指定完整路径
