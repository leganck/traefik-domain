# AGENTS.md

## 构建命令

- `make build` - 构建二进制文件（输出 `traefik-domain`）
- `make clean` - 清理构建产物

## 项目结构

```
main.go          # 入口：轮询 Traefik 并更新 DNS 记录
config/config.go # 通过 viper 加载配置（环境变量 + config.yaml）
dns/             # DNS 提供商实现（dnspod、adguard、cloudflare）
traefik/traefik.go # Traefik API 客户端
```

## 配置

支持 `config.yaml` 和环境变量两种方式。关键配置项：
- `TRAEFIK_HOST` / `traefik.host` - Traefik 地址（支持 `user:pass@host` 认证）
- `DNS_NAME` / `dns.name` - DNS 提供商：`dnspod`、`adguard` 或 `cloudflare`
- `DNS_ID`, `DNS_SECRET` / `dns.id`, `dns.secret` - 提供商凭证
- `POLL_INTERVAL` / `poll.interval` - 轮询间隔（秒）

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
