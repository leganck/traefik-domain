# AGENTS.md

## 构建命令

- `make build` - 构建二进制文件（输出 `traefik-domain`）
- `make clean` - 清理构建产物

## 项目结构

```
main.go               # 入口：独立的 Traefik 和 DNS 轮询
config/runtime.go     # 配置管理和缓存处理
dns/                  # DNS 提供商实现（dnspod、adguard、cloudflare、openwrt）
traefik/traefik.go    # Traefik API 客户端
web/                  # Web UI 处理器和前端
```

## 轮询机制

### 独立轮询

系统采用两个独立的轮询任务：

1. **Traefik 轮询**（默认 30 秒）
   - 从 Traefik API 获取所有路由中的域名
   - 合并到 SwitchConfig（自动保存）
   - 标记域名是否在 Traefik 中（`InTraefik` 字段）

2. **DNS 提供商轮询**（默认 300 秒）
   - 先更新缓存：从 DNS 查询所有记录（包括 Managed 状态）
   - 后同步：根据缓存和开关同步 DNS 记录
   - 自动保存缓存到文件

### 轮询顺序优化

DNS 轮询按以下顺序执行：
1. `updateRecordCache()` - 查询 DNS 记录并更新缓存
2. `syncFromCache()` - 使用最新缓存同步 DNS

这样可以确保每次同步时都有正确的 `Managed` 状态判断。

### 启动初始化

程序启动时会自动执行一次 DNS 轮询，初始化记录缓存，避免首次同步时缓存为空的问题。

## 配置

支持 `config.yaml` 和环境变量两种方式。关键配置项：
- `TRAEFIK_HOST` / `traefik.host` - Traefik 地址（支持 `user:pass@host` 认证）
- `DNS_NAME` / `dns.name` - DNS 提供商：`dnspod`、`adguard`、`cloudflare` 或 `openwrt`
- `DNS_ID`, `DNS_SECRET` / `dns.id`, `dns.secret` - 提供商凭证
- `POLL_INTERVAL` / `poll_interval` - 已废弃（向后兼容）
- `TRAEFIK_POLL_INTERVAL` / `traefik_poll_interval` - Traefik 轮询间隔（秒，默认 30）
- `DNS_POLL_INTERVAL` / `dns_poll_interval` - DNS 提供商轮询间隔（秒，默认 300）

### 配置示例

`data/providers.json`:
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

## Web UI 配置

支持通过 Web 界面管理域名在各 DNS 供应商的同步开关。

环境变量:
- `WEB_ENABLED` / `web.enabled` - 是否启用 Web UI (默认: false)
- `WEB_PORT` / `web.port` - Web 服务端口 (默认: 8080)
- `WEB_CONFIG_PATH` / `web.config-path` - 开关配置文件路径 (默认: ./data/switches.json)

配置示例:
```yaml
web:
  enabled: true
  port: 8080
  config_path: "./data/switches.json"
```

使用说明:
1. 启用 Web UI 后，访问 http://localhost:8080
2. 页面展示从 Traefik 获取的所有域名
3. 每个供应商有全局开关，控制该供应商下所有域名
4. 每个域名可以单独控制各供应商的同步开关
5. 新发现的域名默认所有供应商关闭，需手动开启

## 发布流程

匹配 `v*` 的标签（如 `v1.0.0`）会触发：
1. GoReleaser - 构建多平台二进制文件
2. Docker buildx - 推送到 Docker Hub 和 GHCR

## 注意事项

- 项目中无测试文件
- 使用 CGO_ENABLED=0 进行静态构建
- 根目录已预编译二进制文件：`traefik-domain`
