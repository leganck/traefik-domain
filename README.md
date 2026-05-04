# traefik-domain

自动从 Traefik 发现域名，并按配置同步到指定 DNS 提供商；同时提供 Web UI 用于按域名和按提供商控制同步开关。

## 功能

- 从 Traefik `/api/http/routers` 自动发现启用中的域名
- 支持 DNSPod、AdGuard、Cloudflare、OpenWRT
- 自动识别 A / AAAA / CNAME 记录类型
- 支持 Traefik HTTP Basic Auth
- Traefik 轮询与 DNS 轮询相互独立
- 支持配置文件变更热加载
- 支持 Web UI 管理域名同步开关

## 运行方式

### 1. 准备配置

程序默认读取 `./data/providers.json`，也支持环境变量初始化和覆盖。

### 2. 启动

```bash
make build
./traefik-domain
```

### 3. Docker

示例请参考仓库中的 `example/docker-compose.yml`。

## 配置

### `data/providers.json`

```json
{
  "traefik": {
    "host": "https://traefik.example.com",
    "username": "admin",
    "password": "password"
  },
  "providers": [
    {
      "provider_id": "p_abcd1234",
      "name": "my-adguard",
      "type": "adguard",
      "host": "http://192.168.1.2:3000",
      "id": "admin",
      "secret": "password",
      "record_value": "192.168.1.10"
    }
  ],
  "poll_interval": 5,
  "traefik_poll_interval": 30,
  "dns_poll_interval": 300,
  "web_enabled": true,
  "web_port": 8080,
  "log_level": "info"
}
```

### 环境变量

| 环境变量 | 说明 | 默认值 |
|---|---|---|
| `TRAEFIK_HOST` | Traefik 地址 | - |
| `TRAEFIK_USERNAME` | Traefik 用户名 | - |
| `TRAEFIK_PASSWORD` | Traefik 密码 | - |
| `DNS_NAME` | DNS 提供商名称 | - |
| `DNS_ID` | DNS 提供商 ID / 用户名 | - |
| `DNS_SECRET` | DNS 提供商密钥 / 密码 | - |
| `DNS_RECORD_VALUE` | 记录值，自动识别 A / AAAA / CNAME | - |
| `ADGUARD_HOST` | AdGuard 地址 | - |
| `OPENWRT_HOST` | OpenWRT 地址 | - |
| `POLL_INTERVAL` | 兼容字段，当前仅作为 app 配置保留 | `5` |
| `TRAEFIK_POLL_INTERVAL` | Traefik 轮询间隔（秒） | `30` |
| `DNS_POLL_INTERVAL` | DNS 轮询间隔（秒） | `300` |
| `WEB_ENABLED` | 是否启用 Web UI | `true` |
| `WEB_PORT` | Web 端口 | `8080` |
| `LOG_LEVEL` | 日志级别 | `info` |

## 数据流

```text
main
  -> 读取 providers.json / 环境变量
  -> 监听 providers.json 变更
  -> App 初始化
      -> 加载分文件 state
      -> 初始化 DNS provider 和 Traefik client
      -> 启动 Web UI
      -> 先执行一次 Traefik 轮询
      -> 再执行一次 DNS 轮询

运行中：
  Traefik 轮询 -> 更新 discovery / preferences / records
  DNS 轮询 -> 刷新 DNS 记录缓存
  Web API -> 修改 preferences / provider 开关 -> 立即应用到 DNS
```

## 状态文件

程序把运行状态拆成多个文件，放在 `./data/` 下：

- `domain_preferences.json`
- `domain_discovery.json`
- `domain_records.json`

这些文件由 `DomainSyncState` 统一读写，并在有变更时延迟落盘。

## Web UI

启用后访问 `http://localhost:8080`。

主要 API：

- `GET /api/domains`
- `POST /api/toggle/domain`
- `POST /api/toggle/provider`
- `DELETE /api/domains/{domain}`
- `GET /api/config`
- `PUT /api/config/traefik`
- `GET /api/providers`
- `POST /api/providers`
- `PUT /api/providers/{id}`
- `DELETE /api/providers/{id}`

## 工作原理

1. Traefik 轮询解析 `/api/http/routers`，提取 `Host(...)` 域名
2. 域名写入 `DomainSyncState.Discovery`
3. DNS 轮询从各 provider 拉取记录，写入 `DomainSyncState.Records`
4. Web/UI 修改 `DomainSyncState.Preferences`
5. `DNSManager.Apply()` 根据开关执行 `EnsureDomain()` 或 `DeleteManagedDomain()`
6. 状态会在后台自动 flush 到磁盘

## 项目结构

```text
cmd/traefik-domain/main.go   # 入口
config/                      # providers.json、环境变量、热加载
internal/app/                # 应用装配、轮询、重载
internal/service/            # DNS 管理器
internal/state/              # 域名偏好、发现、记录缓存与持久化
dns/                         # DNS provider 适配层
traefik/                     # Traefik API 客户端
web/                         # Web API 和静态资源
example/                     # 示例配置
```

## 构建

```bash
make build
make clean
```
