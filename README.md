# sing-box-easy

sing-box-easy 是一个带有现代化 Web 界面的 sing-box 配置管理工具。它提供功能完整的仪表盘，用于可视化管理 sing-box 的配置、节点、订阅与服务生命周期。

## ✨ 主要特性

### 🖥️ Web 仪表盘
- **可视化配置管理**：内置 Monaco Editor，支持 JSON 语法高亮、补全与校验。 版本管理
[![Editor](./doc/images/editor.png)]
[![Editor](./doc/images/version.png)]
- **节点与订阅**：直观的节点列表、分组（selector / urltest）编辑、订阅自动更新调度器。
[![Editor](./doc/images/subscriptions.png)]
[![Editor](./doc/images/filters.png)]
- **初始化向导**：首次启动自动进入 `/init` 引导，分步完成 sing-box 安装、配置生成、Dashboard UI 部署。
- **实时监控**：查看 sing-box 进程状态与日志。
[![Editor](./doc/images/logs.png)]
- **现代化 UI**：Vue 3 + TypeScript + TailwindCSS v4 + DaisyUI + PrimeVue。

### ⚙️ 后端核心
- **RESTful API**：覆盖配置、DNS、Inbound/Outbound、路由、订阅、调度器、安装、Dashboard、初始化等共 79 个端点。
- **配置安全**：每次写盘都走 `写临时文件 → sing-box check → 原子替换 → 备份` 流程，可一键 `/config/rollback` 回滚。
- **引用一致性**：删除 outbound 或订阅刷新时自动清理 `selector` / `urltest` 中已失效的引用，避免 sing-box 静默运行时悬挂指针。
- **多协议解析**：Shadowsocks (`ss://`)、VMess (`vmess://`)、Trojan (`trojan://`)。
- **统一响应信封**：所有业务接口返回 `{ code, data, msg }`，HTTP 状态码恒为 200，前端按业务 `code` 分流。
- **订阅自动更新**：cron 调度器（默认每 5 分钟）按订阅配置的 `update_interval` 增量拉取、差分应用。
- **高性能存储**：SQLite + XORM（`modernc.org/sqlite` 纯 Go，无需 CGO）。

## 🚀 安装

> 目前仅支持 **Debian 系（含 Ubuntu）x86_64** 系统，且需预先安装 [`sing-box` 内核](https://sing-box.sagernet.org/installation/package-manager/)。

脚本会自动检测系统与架构、下载最新 Release、解压到当前目录、生成 `app.yml`，并以 systemd 服务方式启动 `sing-box-easy`（无 systemd 时回退到后台进程），最后做一次健康检查。

```bash
# 下载并运行安装脚本
curl -fsSLO https://raw.githubusercontent.com/SealinGp/sing-box-easy/main/scripts/install.sh
bash install.sh
```

若未在默认路径 `/etc/sing-box/config.json` 找到 sing-box 配置，脚本会提示你输入 `config.json` 的完整路径。

常用可选项：

```bash
# 安装指定版本（默认安装最新 Release）
bash install.sh v1.0.0

# 预设 sing-box 配置路径，跳过交互提示（适合非交互 / 管道执行）
SINGBOX_CONFIG=/etc/sing-box/config.json bash install.sh

# 自定义监听端口 / 安装目录
PORT=9090 INSTALL_DIR=/opt/sing-box-easy bash install.sh
```

安装完成后通过 systemd 管理服务：

```bash
sudo systemctl status  sing-box-easy
sudo systemctl restart sing-box-easy
sudo journalctl -u sing-box-easy -f
```
