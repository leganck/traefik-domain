# traefik-domain

自动从 Traefik 获取域名路由信息，并在指定的 DNS 提供商处添加或更新 DNS 记录。

## 功能特性

- 自动发现 Traefik 中启用的域名路由
- 支持多种 DNS 提供商：**DNSPod**、**AdGuard**、**Cloudflare**、**OpenWRT**
- 支持 A（IPv4）、AAAA（IPv6）、CNAME（域名）记录类型，自动检测
- 支持 Traefik HTTP Basic Auth 认证
- 定时轮询，自动同步

## 快速开始

### Docker Compose（推荐）

```yaml
version: "3.8"

services:
  traefik-domain:
    image: leganck/traefik-domain:latest
    container_name: traefik-domain
    restart: unless-stopped
    environment:
      - TRAEFIK_HOST=http://traefik:8080
      - POLL_INTERVAL=5
      - DNS_NAME=dnspod
      - DNS_ID=your_dns_id
      - DNS_SECRET=your_dns_secret
      - DNS_RECORD_VALUE=192.168.1.1
```

更多示例请参考 [docker-compose.yml](docker-compose.yml)。

### Docker Run

```bash
docker run -d \
  --name traefik-domain \
  --restart unless-stopped \
  -e TRAEFIK_HOST=http://traefik:8080 \
  -e POLL_INTERVAL=5 \
  -e DNS_NAME=dnspod \
  -e DNS_ID=your_dns_id \
  -e DNS_SECRET=your_dns_secret \
  -e DNS_RECORD_VALUE=192.168.1.1 \
  leganck/traefik-domain:latest
```

### 二进制运行

```bash
# 从 GitHub Releases 下载对应平台的二进制文件
# https://github.com/leganck/traefik-domain/releases

# 使用环境变量
export TRAEFIK_HOST=http://traefik:8080
export DNS_NAME=dnspod
export DNS_ID=your_dns_id
export DNS_SECRET=your_dns_secret
export DNS_RECORD_VALUE=192.168.1.1
./traefik-domain

# 或使用配置文件
./traefik-domain  # 自动加载当前目录下的 config.yaml
```

### 源码编译

```bash
git clone https://github.com/leganck/traefik-domain.git
cd traefik-domain
make build
```

## 配置说明

支持 **环境变量** 和 **config.yaml** 配置文件两种方式，环境变量优先。

### 环境变量

| 环境变量 | 配置文件键 | 说明 | 默认值 |
|---|---|---|---|
| `TRAEFIK_HOST` | `traefik.host` | Traefik API 地址，支持 `user:pass@host` 认证格式 | - |
| `POLL_INTERVAL` | `poll.interval` | 轮询间隔（秒） | `5` |
| `DNS_NAME` | `dns.name` | DNS 提供商：`dnspod`、`adguard`、`cloudflare`、`openwrt` | - |
| `DNS_ID` | `dns.id` | DNS 提供商认证 ID / 用户名 | - |
| `DNS_SECRET` | `dns.secret` | DNS 提供商认证密钥 / 密码 | - |
| `DNS_RECORD_VALUE` | `dns.record.value` | DNS 记录值，支持 IPv4、IPv6 或域名 | - |
| `DNS_REFRESH` | `dns.refresh` | 是否强制刷新 DNS 记录 | `false` |
| `AD_GUARD_HOST` | `adguard.host` | AdGuard Home 地址（`dns.name=adguard` 时必填） | - |
| `OPENWRT_HOST` | `openwrt.host` | OpenWRT 地址（`dns.name=openwrt` 时必填） | - |
| `LOG_LEVEL` | `log.level` | 日志级别：`debug`、`info`、`warn`、`error` | `info` |

### config.yaml 示例

```yaml
traefik:
  host: "http://traefik:8080"

poll:
  interval: 5

dns:
  name: "dnspod"
  id: "your_dns_id"
  secret: "your_dns_secret"
  record:
    value: "192.168.1.1"
  refresh: false

adguard:
  host: "http://adguard:3000"

log:
  level: "info"
```

配置文件搜索路径：`./config.yaml` → `./config/config.yaml` → `/etc/traefik-domain/config.yaml` → `~/.traefik-domain/config.yaml`

### DNS_RECORD_VALUE 说明

根据值的类型自动检测记录类型：

| 值类型 | 示例 | 记录类型 |
|---|---|---|
| IPv4 地址 | `192.168.1.1` | A |
| IPv6 地址 | `2001:db8::1` | AAAA |
| 域名 | `server.example.com` | CNAME |

## DNS 提供商配置示例

### DNSPod

```yaml
dns:
  name: "dnspod"
  id: "your_dnspod_id"
  secret: "your_dnspod_token"
  record:
    value: "192.168.1.1"
```

### AdGuard

```yaml
dns:
  name: "adguard"
  id: "admin"
  secret: "password"
  record:
    value: "192.168.1.1"

adguard:
  host: "http://adguard:3000"
```

### Cloudflare

```yaml
dns:
  name: "cloudflare"
  id: "your_api_token"
  secret: "your_zone_id"
  record:
    value: "server.example.com"
```

### OpenWRT

```yaml
dns:
  name: "openwrt"
  id: "root"
  secret: "password"
  record:
    value: "192.168.1.1"

openwrt:
  host: "http://openwrt:80"
```

## Traefik 认证

`TRAEFIK_HOST` 支持在 URL 中嵌入 HTTP Basic Auth 凭证：

```bash
# 无认证
TRAEFIK_HOST=http://traefik:8080

# HTTP Basic Auth
TRAEFIK_HOST=admin:secretpassword@traefik:8080
```

## 项目结构

```
.
├── main.go                # 入口：轮询 Traefik 并更新 DNS 记录
├── config/config.go       # 通过 viper 加载配置（环境变量 + config.yaml）
├── dns/
│   ├── dns.go             # DNS Provider 接口与调度逻辑
│   ├── model/             # DNS 记录模型
│   └── provider/
│       ├── dnspod.go      # DNSPod 实现
│       ├── adguard.go     # AdGuard 实现
│       ├── cloudflare.go  # Cloudflare 实现
│       └── openwrt.go     # OpenWRT 实现
├── traefik/traefik.go     # Traefik API 客户端
├── util/                  # 工具函数
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── .goreleaser.yml
```

## 工作原理

```
┌──────────────┐     轮询      ┌──────────────┐    添加/更新    ┌──────────────┐
│              │ ───────────→  │              │ ─────────────→  │              │
│   Traefik    │   API 获取    │ traefik-     │   DNS 记录      │  DNS Provider│
│   API        │   域名路由    │  domain      │                 │ (DNSPod/CF/  │
│              │ ←───────────  │              │ ←─────────────  │  AdGuard/OW) │
└──────────────┘               └──────────────┘                 └──────────────┘
```

1. 定时从 Traefik API (`/api/http/routers`) 获取所有已启用的路由
2. 解析路由规则中的 `Host()` 域名
3. 按 DNS 提供商进行域名拆分（主域名 + 子域名）
4. 查询 DNS 提供商现有记录
5. 新增缺失的记录，更新值不匹配的记录

## Docker 镜像

| Registry | 镜像地址 |
|---|---|
| Docker Hub | `leganck/traefik-domain` |
| GHCR | `ghcr.io/leganck/traefik-domain` |

支持架构：`linux/amd64`、`linux/arm`、`linux/arm64`

## 发布流程

匹配 `v*` 的 Git 标签会自动触发：

1. **GoReleaser** - 构建多平台二进制文件并发布到 GitHub Releases
2. **Docker buildx** - 构建多架构 Docker 镜像并推送到 Docker Hub 和 GHCR

```bash
git tag v1.0.0
git push origin v1.0.0
```

## License

MIT
