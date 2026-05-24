# AGENTS.md

## 构建命令

- `make build` - 构建二进制文件（输出 `traefik-domain`）
- `make clean` - 清理构建产物

## 运行时结构

```
cmd/traefik-domain/main.go   # 入口：加载 providers 配置、启动 watcher、运行 App
config/                      # providers.json、环境变量、热加载
internal/app/                # App 装配、Traefik/DNS 轮询、Web UI、重载
internal/service/            # DNSManager：把 Web 操作转换为 DNS 变更
internal/state/              # DomainSyncState：偏好、发现、记录缓存、持久化
dns/                         # DNS 提供商实现（dnspod、adguard、cloudflare、openwrt）
traefik/traefik.go           # Traefik API 客户端
web/                         # Web API 处理器
```

## 数据流转

### 启动

1. `main.go` 读取 `data/providers.json`
2. 启动文件 watcher，监听配置变更
3. 创建 `App`
4. `App.Run()` 初始化状态、DNS manager、Traefik client 和 Web UI
5. 启动后先执行一次 Traefik 轮询，再执行一次 DNS 轮询

### Traefik 轮询

1. `App.PollTraefik()` 调用 `traefik.Client.Domains()`
2. 从 `/api/http/routers` 中提取 `Host(...)` 域名
3. `DomainSyncState.MergeDomains()` 更新：
   - `Discovery.InTraefik`
   - 补齐 `Preferences`
   - 补齐 `Records`
4. 状态通过延迟 flush 写入磁盘

### DNS 轮询

1. `App.PollDNS()` 调用 `DNSManager.RefreshAllStates()`
2. `DNSManager` 遍历每个 provider
3. `refreshProviderRecordState()` 查询各主域名的记录
4. `DomainSyncState.UpdateRecords()` 更新记录缓存，并清理已失效的 override

### Web 写入

1. Web API 修改 `DomainSyncState.Preferences` 或 `ProviderGlobals`
2. `App.applyDomainUpdates()` 把变更转换成 `DNSJob`
3. `DNSManager.Apply()` 执行：
   - `EnsureDomain()`
   - `DeleteManagedDomain()`
4. 执行后再次刷新 provider record cache

## 配置

当前运行配置以 `config/providers.go` 为准，关键项如下：

- `TRAEFIK_HOST` / `traefik.host` - Traefik 地址
- `TRAEFIK_USERNAME` / `traefik.username` - Traefik 用户名
- `TRAEFIK_PASSWORD` / `traefik.password` - Traefik 密码
- `DNS_NAME` / `providers[].type` - DNS 提供商：`dnspod`、`adguard`、`cloudflare`、`openwrt`
- `DNS_ID`, `DNS_SECRET` - 提供商凭证
- `DNS_RECORD_VALUE` - 记录值，自动识别 A / AAAA / CNAME
- `ADGUARD_HOST` - AdGuard 地址
- `OPENWRT_HOST` - OpenWRT 地址
- `POLL_INTERVAL` - 兼容字段，当前仍保留但不是主流程重点
- `TRAEFIK_POLL_INTERVAL` - Traefik 轮询间隔（秒，默认 30）
- `DNS_POLL_INTERVAL` - DNS 轮询间隔（秒，默认 300）
- `WEB_ENABLED` - 是否启用 Web UI（默认 true）
- `WEB_PORT` - Web 端口（默认 8080）
- `LOG_LEVEL` - 日志级别

### 配置示例

`data/providers.json`：

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
  "traefik_poll_interval": 30,
  "dns_poll_interval": 300,
  "web_enabled": true,
  "web_port": 8080,
  "log_level": "info"
}
```

## Web UI

- 启用后访问 `http://localhost:8080`
- 页面展示 Traefik 发现的域名、provider、开关和记录缓存
- 支持域名级和 provider 级开关
- 删除域名前会检查该域名是否仍在 Traefik 中

## 状态文件

`internal/state/` 使用分文件持久化：

- `./data/domain_preferences.json`
- `./data/domain_discovery.json`
- `./data/domain_records.json`

`DomainSyncState` 负责读写这些文件，变更后延迟写盘，退出时会 `Flush()`。

## 备注

- 项目当前没有测试文件
- 使用 `CGO_ENABLED=0` 进行静态构建
- 根目录存在预编译二进制：`traefik-domain`

