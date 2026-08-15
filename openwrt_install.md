# 在 OpenWrt 上安装 sing-box-easy

适用于 OpenWrt 及其衍生版（ImmortalWrt、LEDE、Lean 等）。安装包为 `.ipk`，通过 `opkg` 安装。

> 本文假设这是一次**全新安装**。如果路由器上已经装了 OpenClash / homeproxy / PassWall 等代理插件，请先读 [0. 与其他代理插件共存](#0-与其他代理插件共存)。

---

## 0. 与其他代理插件共存

sing-box-easy 只负责**管理 sing-box 的配置和进程**，它不会自己创建防火墙规则、不会改 dnsmasq、也不会接管数据面。数据面由你在向导里配置的 sing-box **入站（TUN / tproxy）**决定。

这意味着：**同一时间只能有一个插件接管流量**。OpenClash、homeproxy、PassWall 都会安装自己的 nftables 规则并劫持 DNS，和 sing-box 的 TUN 一起跑必定冲突。

如果你已经装了这些插件：

```sh
# 1. 先只“停用”，不要卸载 —— 这是你的回滚路径
/etc/init.d/openclash stop  && /etc/init.d/openclash disable
/etc/init.d/homeproxy stop  && /etc/init.d/homeproxy disable
/etc/init.d/passwall  stop  && /etc/init.d/passwall  disable

# 2. 确认数据面已经干净
nft list table inet fw4 | grep -ciE "openclash|passwall|homeproxy"   # 期望 0
ip link show utun 2>/dev/null || echo "utun 已移除"

# 3. 确认 dnsmasq 没有被指向已停止的插件
uci show dhcp.@dnsmasq[0] | grep -E "server|noresolv"
#   若指向 127.0.0.1#7874 之类的插件端口，需要改回正常上游，否则整个局域网会没有 DNS：
#   uci -q delete dhcp.@dnsmasq[0].server
#   uci set dhcp.@dnsmasq[0].noresolv='0'
#   uci commit dhcp && /etc/init.d/dnsmasq restart
```

确认 sing-box-easy 工作正常后，再决定是否 `opkg remove`。

> ⚠️ **不要把这些插件生成的 `config.json` 拿来当 sing-box-easy 的初始配置。**
> homeproxy / PassWall 生成的配置里带有大量它们自己的约定（写死的 `/var/run/homeproxy/` 日志路径、`dns_direct` 之类的 DNS 服务器标签、redirect/tproxy 入站、指向插件目录的 `external_ui`）。这些引用在插件被停用后全部失效，会让初始化向导报出很难懂的错误，例如：
>
> ```
> initialize outbound[21]: default domain resolver not found: dns_direct
> ```
>
> 让向导从空白配置开始，然后导入你自己的订阅。旧配置留个备份即可：
>
> ```sh
> cp /etc/sing-box/config.json /root/config.json.bak
> ```

---

## 1. 前置条件

| 项目 | 说明 |
| --- | --- |
| OpenWrt | 21.02 及以上（含 ImmortalWrt 等衍生版） |
| 磁盘空间 | ≥ 60 MB 可用 overlay（程序本身约 41 MB） |
| sing-box 内核 | 需要单独安装，见下 |
| `kmod-tun` | 使用 TUN 入站时必需 |

安装 sing-box 内核与依赖：

```sh
opkg update
opkg install sing-box kmod-tun ca-bundle
sing-box version
```

> 面板的 API 版本对应 sing-box **1.12.x**。请尽量使用软件源里的版本，不要装 beta——1.13 起 DNS 配置结构有较大变化，可能与面板生成的配置不兼容。

---

## 2. 选择正确的 ipk

先查看架构：

```sh
opkg print-architecture
# 或
. /etc/openwrt_release && echo "$DISTRIB_ARCH"
```

| `DISTRIB_ARCH` | 下载文件 | 常见设备 |
| --- | --- | --- |
| `x86_64` | `sing-box-easy_<ver>_x86_64.ipk` | 软路由、虚拟机、N100 之类 |
| `aarch64_generic` | `sing-box-easy_<ver>_aarch64_generic.ipk` | 树莓派 4/5、多数 ARM64 路由器 |
| `arm_cortex-a7` | `sing-box-easy_<ver>_arm_cortex-a7.ipk` | 部分 ARMv7 路由器 |

> 架构不在表里时，请在 [Releases](https://github.com/SealinGp/sing-box-easy/releases) 页面确认是否有对应包。

---

## 3. 下载、校验并安装

> 若当前是通过代理插件才能访问 GitHub，请**在停用旧插件之前**先把 ipk 下载好。

```sh
cd /tmp
VER=1.2.5        # 改成 Releases 页面的最新版本
ARCH=x86_64      # 改成上一步查到的架构

curl -fLO "https://github.com/SealinGp/sing-box-easy/releases/download/v${VER}/sing-box-easy_${VER}_${ARCH}.ipk"
curl -fL  "https://github.com/SealinGp/sing-box-easy/releases/download/v${VER}/sing-box-easy_${VER}_${ARCH}.ipk.sha256"

# 核对哈希（注意：校验文件里记录的是构建时的路径，直接 sha256sum -c 会失败，请人工比对）
sha256sum "sing-box-easy_${VER}_${ARCH}.ipk"

opkg install "/tmp/sing-box-easy_${VER}_${ARCH}.ipk"
```

安装脚本会自动 `enable` 并 `start` 服务。**注意：这一步只启动面板本身，不会启动 sing-box，也不会改动任何网络配置。**

验证：

```sh
netstat -lntu | grep 8080
logread | grep 'sing-box-easy\[' | tail -5
```

然后浏览器打开 **`http://<路由器IP>:8080`**。

- OpenWrt 上默认**不需要登录**（`server.auth: auto` 判定为可信局域网），界面使用顶栏布局以免和 LuCI 的侧边栏重复。
- LuCI 里也会出现入口：**服务 → sing-box-easy**。若没看到，执行 `rm -f /tmp/luci-indexcache*` 后刷新页面。

---

## 4. 初始化向导：OpenWrt 推荐值

首次打开会进入初始化向导。以下是在 OpenWrt 上的推荐填法。

| 步骤 | 推荐值 | 原因 |
| --- | --- | --- |
| **1 安装 sing-box** | 跳过（已用 opkg 装好） | beta 选项在 OpenWrt 上不支持 |
| **2 日志** | 级别 `info`；**输出路径留空** | 留空才会写到 syslog，面板的日志页通过 `logread` 读取。填了文件路径而该文件不存在时，日志页会一直是空的。需要用 DNS 诊断的规则归因时临时改成 `debug` |
| **3 Clash API** | `0.0.0.0:9090`；外部 UI `/etc/sing-box/ui`；缓存 `/etc/sing-box/cache.db` | 端口若已被其他插件占用请换一个（如 `9092`）。缓存要放持久化目录：它保存节点选择和**规则集缓存**，开机时无需联网即可启动 |
| **4 面板 UI** | 保持默认 | 目录不存在时 sing-box 会自行下载 |
| **5 出站** | 粘贴订阅链接 | 会自动记录到「订阅管理」并由定时任务刷新 |
| **6 规则集** | 保持默认 | 默认已走 gh-proxy 镜像，直连 GitHub 在国内通常不可达 |
| **7 DNS** | 国内 `223.5.5.5`；国外走代理的 DoT/DoH | 广告拦截请用规则动作 `predefined` + `NXDOMAIN`，**没有 `block` 类型的 DNS 服务器** |
| **8 入站** | **TUN**，`auto_route` + `strict_route` 开启，地址改成 **`172.16.250.1/30`** | ⚠️ 预设的 `172.19.0.1/30` 会和 Docker 默认网段（`172.17.0.0/16`–`172.28.0.0/16`）冲突。装了 Docker 的软路由必须改 |
| **9 路由** | 第一条规则加 `ip_is_private → direct`；最终出站指向你的节点组 | 没有这条规则时，局域网和 Docker 网段的流量会被送进代理 |

完成后**先不要启动 sing-box**，请继续读下一节。

---

## 5. 启动前的检查

sing-box 在**启动时**会拉取所有 `remote` 类型的规则集，任何一个失败都会导致启动失败（`sing-box check` 不会发现这个问题，因为它不联网）：

```
FATAL start service: initialize rule-set[24]: geosite-category-ads-all:
  Get "https://raw.githubusercontent.com/...": i/o timeout
```

这在切换代理插件时特别危险：旧插件已停、新的还没起来，此时没有任何可用代理去下载规则集，结果就是**整个局域网既没有代理也没有网**。

两个稳妥做法，二选一：

**A. 规则集全部改成本地文件（最稳）**

```sh
mkdir -p /etc/sing-box/rule-sets
# 趁网络还通的时候，把每个规则集下载下来
curl -fL -o /etc/sing-box/rule-sets/geosite-cn.srs \
  "https://gh-proxy.com/https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs"
# …其余同理
```

然后在面板的「路由 → 规则集」里把类型改成 `local`、路径指向上面的文件。这样启动完全不依赖网络。

**B. 保持 `remote`，但确保下载可达**

- URL 统一加 `https://gh-proxy.com/` 前缀；
- `download_detour` 设为**直连**出站，而不是某个可能不通的节点；
- 开启 `cache_file`，首次成功后会缓存，后续启动不再依赖网络。

确认无误后再启动：

```sh
sing-box check -c /etc/sing-box/config.json && echo OK
```

然后在面板上点「启动」，或：

```sh
/etc/init.d/sing-box start
logread | grep 'sing-box\[' | tail -20
```

---

## 6. 文件路径与端口

### 路径一览

| 内容 | 路径 |
| --- | --- |
| **面板配置（端口、认证、日志级别）** | `/etc/sing-box-easy/app.yml` |
| 配置示例 | `/etc/sing-box-easy/app.yml.example` |
| 面板程序 | `/usr/bin/sing-box-easy` |
| 面板服务脚本 | `/etc/init.d/sing-box-easy` |
| LuCI 菜单端口配置 | `/etc/config/sing-box-easy` |
| **sing-box 配置** | `/etc/sing-box/config.json` |
| 面板数据库（订阅、设置、版本历史） | `/etc/sing-box/sing-box-easy.db` |
| sing-box 缓存 | `/etc/sing-box/cache.db` |
| Clash 面板静态文件 | `/etc/sing-box/ui/` |

`app.yml` 和 `/etc/config/sing-box-easy` 都是 opkg 的 **conffiles**，升级时不会被覆盖。

### 默认端口

**`8080`**，定义在 `/etc/sing-box-easy/app.yml`：

```yaml
server:
  port: "8080"
```

### 修改端口

```sh
# 1. 改面板配置
vi /etc/sing-box-easy/app.yml          # 把 server.port 改成新端口，例如 "8081"

# 2. 同步 LuCI 菜单的跳转端口
uci set sing-box-easy.main.port='8081'
uci commit sing-box-easy

# 3. 重启面板
/etc/init.d/sing-box-easy restart

# 4. 确认
netstat -lntu | grep 8081
```

> 也可以用环境变量 `HTTP_PORT` 临时覆盖，但通过 procd 启动时不方便传递，改配置文件更可靠。

---

## 7. 常用命令

```sh
# 面板（sing-box-easy 本身）
/etc/init.d/sing-box-easy start | stop | restart | enable | disable
logread | grep 'sing-box-easy\['

# 代理内核（sing-box）—— 也可以在面板上操作
/etc/init.d/sing-box start | stop | restart
logread | grep 'sing-box\['

# 配置校验
sing-box check -c /etc/sing-box/config.json
```

> `/etc/init.d/sing-box` 是 UCI 驱动的：它会读 `/etc/config/sing-box`，当 `enabled` 为 `0` 时**直接返回成功但什么都不做**。若面板提示启动成功而服务并未运行，先检查：
>
> ```sh
> uci get sing-box.main.enabled     # 应为 1
> uci get sing-box.main.conffile    # 应为 /etc/sing-box/config.json
> ```

---

## 8. 升级

ipk 安装的实例**不能用面板的自动更新**：文件归 opkg 管理，直接替换会让 opkg 的文件记录错乱。面板会识别这一点，把「更新」按钮变成「准备安装包」——它会下载并校验好对应架构的 ipk，然后给出命令让你自己执行：

```sh
opkg install /tmp/sing-box-easy_<新版本>_<架构>.ipk
```

之所以不由面板自动执行：ipk 的 `prerm` 会先停掉 sing-box-easy 本身，如果由面板进程发起安装，会在事务中途把自己杀掉。

---

## 9. 卸载

```sh
/etc/init.d/sing-box-easy stop
opkg remove sing-box-easy

# 配置和数据不会被自动删除，需要时手动清理：
rm -rf /etc/sing-box-easy
rm -f  /etc/sing-box/sing-box-easy.db
```

---

## 10. 故障排查

| 现象 | 排查方向 |
| --- | --- |
| 装完访问不了 8080 | `netstat -lntu \| grep 8080`；`logread \| grep 'sing-box-easy\['` |
| 安装时报 `file_sha256sum_alloc: Failed to open file .../app.yml` | 1.2.5 及更早版本的已知问题，不影响安装，升级后消失 |
| 面板显示已启动、但服务没起来 | 看第 7 节的 UCI `enabled` 说明 |
| 日志页面空白 | `log.output` 应留空，让日志进 syslog |
| sing-box 启动即 FATAL | 多为规则集下载失败，见第 5 节 |
| 局域网能上网但代理不生效 | 检查 TUN 是否起来（`ip link show tun0`）、`auto_route` 是否开启 |
| 局域网互访异常 / Docker 容器访问不了 | 缺少 `ip_is_private → direct` 路由规则，或 TUN 网段和 Docker 冲突 |
| LuCI 里没有菜单入口 | `rm -f /tmp/luci-indexcache*` 后刷新 |

---

相关文档：[Debian / Ubuntu 安装](debian_install.md) · [返回 README](README.md)
