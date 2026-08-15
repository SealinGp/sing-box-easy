# 在 Debian / Ubuntu 上安装 sing-box-easy

适用于 Debian 系发行版（Debian、Ubuntu 及其衍生版）。安装方式为发行包（tar.gz）+ systemd 服务。

---

## 1. 前置条件

| 项目 | 说明 |
| --- | --- |
| 系统 | Debian 11+ / Ubuntu 20.04+，x86_64 或 arm64 |
| 权限 | 需要 root（或 sudo）以安装 systemd 服务 |
| sing-box 内核 | 需要单独安装，见下 |
| 依赖 | `curl`、`tar`（通常已自带） |

程序本身是静态编译的单文件，不依赖 libc 版本，也不需要 CGO / SQLite 运行库。

### 安装 sing-box 内核

按官方文档用包管理器安装：

```bash
# 官方源（推荐）
curl -fsSL https://sing-box.app/install.sh | sh

sing-box version
```

> 面板的 API 版本对应 sing-box **1.12.x**。1.13 起 DNS 配置结构有较大变化，可能与面板生成的配置不兼容。

---

## 2. 一键安装（推荐）

脚本会自动识别系统与架构、下载最新 Release、解压到当前目录、生成 `app.yml`，并注册 systemd 服务（无 systemd 时回退为后台进程），最后做一次健康检查。

```bash
curl -fsSLO https://raw.githubusercontent.com/SealinGp/sing-box-easy/main/scripts/install.sh
bash install.sh
```

若在默认路径 `/etc/sing-box/config.json` 找不到 sing-box 配置，脚本会提示你输入完整路径。

常用可选项：

```bash
# 安装指定版本（默认最新 Release）
bash install.sh v1.2.5

# 预设 sing-box 配置路径，跳过交互（适合非交互 / 管道执行）
SINGBOX_CONFIG=/etc/sing-box/config.json bash install.sh

# 自定义端口与安装目录
PORT=9090 INSTALL_DIR=/opt/sing-box-easy bash install.sh
```

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `INSTALL_DIR` | 当前目录 | 程序、`app.yml`、`dist/` 的存放位置 |
| `PORT` | `8080` | 面板监听端口 |
| `SINGBOX_CONFIG` | `/etc/sing-box/config.json` | sing-box 配置路径 |

> 已存在的 `app.yml` **不会**被覆盖，升级时你的设置会保留。

---

## 3. 手动安装

不想用脚本时：

```bash
VER=1.2.5
ARCH=amd64        # 或 arm64 / arm
INSTALL_DIR=/opt/sing-box-easy

mkdir -p "$INSTALL_DIR" && cd "$INSTALL_DIR"
curl -fLO "https://github.com/SealinGp/sing-box-easy/releases/download/v${VER}/sing-box-easy-linux-${ARCH}.tar.gz"
curl -fL  "https://github.com/SealinGp/sing-box-easy/releases/download/v${VER}/sing-box-easy-linux-${ARCH}.tar.gz.sha256"
sha256sum -c "sing-box-easy-linux-${ARCH}.tar.gz.sha256"

tar -xzf "sing-box-easy-linux-${ARCH}.tar.gz"     # 解出 sing-box-easy、dist/、app.yml.example
cp app.yml.example app.yml
vi app.yml                                         # 按需修改端口和 sing-box 配置路径
```

注册 systemd 服务：

```bash
cat >/etc/systemd/system/sing-box-easy.service <<EOF
[Unit]
Description=sing-box-easy management panel
After=network.target

[Service]
Type=simple
# WorkingDirectory 必须是安装目录：程序会在此解析 dist/ 前端资源
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/sing-box-easy -c ${INSTALL_DIR}/app.yml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now sing-box-easy
```

---

## 4. 访问面板

浏览器打开 **`http://<服务器IP>:8080`**。

- 非 OpenWrt 系统上**默认要求登录**，初始账号密码为 `admin` / `admin`，**首次登录后请立即修改**。
- 若部署在公网服务器上，请务必在云厂商安全组和本机防火墙上限制该端口的来源，或只监听内网地址。

初始化向导会引导你完成日志、Clash API、出站订阅、规则集、DNS、入站和路由的配置。

> 与 OpenWrt 不同，Debian 上通常用 **TUN** 或 **mixed（HTTP/SOCKS）** 入站。若只是给本机用，`mixed` 入站最简单；若要给整个网络做网关，才需要 TUN + `auto_route`。

---

## 5. 文件路径与端口

### 路径一览

以脚本安装（`INSTALL_DIR` 默认为执行目录）为例：

| 内容 | 路径 |
| --- | --- |
| **面板配置（端口、认证、日志级别）** | `${INSTALL_DIR}/app.yml` |
| 配置示例 | `${INSTALL_DIR}/app.yml.example` |
| 面板程序 | `${INSTALL_DIR}/sing-box-easy` |
| 前端静态资源 | `${INSTALL_DIR}/dist/`（同时也内嵌在程序里，磁盘上的优先） |
| systemd 服务 | `/etc/systemd/system/sing-box-easy.service` |
| **sing-box 配置** | `/etc/sing-box/config.json` |
| 面板数据库（订阅、设置、版本历史） | `/etc/sing-box/sing-box-easy.db` |

具体路径以 `app.yml` 中的 `sing_box.config_path` 和 `sing_box.database_path` 为准。

### 默认端口

**`8080`**，定义在 `app.yml`：

```yaml
server:
  port: "8080"
```

### 修改端口

```bash
# 1. 改配置
vi ${INSTALL_DIR}/app.yml          # server.port 改成新端口，例如 "8081"

# 2. 重启服务
sudo systemctl restart sing-box-easy

# 3. 确认
ss -lntp | grep 8081
```

也可以用环境变量覆盖，适合临时测试或容器场景：

```bash
sudo systemctl edit sing-box-easy
# 加入：
#   [Service]
#   Environment=HTTP_PORT=8081
sudo systemctl restart sing-box-easy
```

---

## 6. 常用命令

```bash
# 面板（sing-box-easy 本身）
sudo systemctl status  sing-box-easy
sudo systemctl restart sing-box-easy
sudo systemctl stop    sing-box-easy
sudo journalctl -u sing-box-easy -f

# 代理内核（sing-box）—— 也可以在面板上操作
sudo systemctl status  sing-box
sudo journalctl -u sing-box -f

# 配置校验
sing-box check -c /etc/sing-box/config.json
```

---

## 7. 升级

Debian 上支持**面板内自动更新**：设置 → 关于 → 应用更新，选择版本后点更新。面板会下载发行包、校验 sha256、原子替换程序与 `dist/`，然后重启自身；失败会自动回滚。

也可以用脚本：

```bash
curl -fsSLO https://raw.githubusercontent.com/SealinGp/sing-box-easy/main/scripts/update.sh
bash update.sh
```

> 提示：未登录 GitHub 时，检查更新受 **每 IP 每小时 60 次** 的匿名限制。在「设置 → GitHub 账号」里填入 OAuth App 的 Client ID 并登录后可提升到每小时 5000 次。Client ID 保存在数据库中，无需改配置文件、也不用重启。

---

## 8. 卸载

```bash
sudo systemctl disable --now sing-box-easy
sudo rm -f /etc/systemd/system/sing-box-easy.service
sudo systemctl daemon-reload

# 程序与配置
rm -rf ${INSTALL_DIR}

# 数据库（如不再需要）
sudo rm -f /etc/sing-box/sing-box-easy.db
```

---

## 9. 故障排查

| 现象 | 排查方向 |
| --- | --- |
| 服务起不来 | `journalctl -u sing-box-easy -n 50` |
| 页面打得开但一片空白 | `dist/` 缺失且程序为旧版（新版前端已内嵌）；确认 `WorkingDirectory` 指向安装目录 |
| 提示找不到 sing-box | `app.yml` 的 `sing_box.binary_path` 留空表示走 `PATH`；否则填绝对路径 |
| 保存配置报校验失败 | 错误信息来自 `sing-box check`，按提示修正；面板不会写入无法通过校验的配置 |
| sing-box 启动即失败 | 常见于 `remote` 规则集在启动时下载失败——它是致命错误。可改用本地规则集文件，或确认下载出站可用 |
| 检查更新报 403 | GitHub 匿名限流，见第 7 节 |

---

相关文档：[OpenWrt 安装](openwrt_install.md) · [返回 README](README.md)
