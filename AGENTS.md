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

## 发布流程

匹配 `v*` 的标签（如 `v1.0.0`）会触发：
1. GoReleaser - 构建多平台二进制文件
2. Docker buildx - 推送到 Docker Hub 和 GHCR

## 注意事项

- 项目中无测试文件
- 使用 CGO_ENABLED=0 进行静态构建
- 根目录已预编译二进制文件：`traefik-domain`
