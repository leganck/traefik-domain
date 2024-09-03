# Docker Traefik Domain

## 项目简介
该项目是一个用于自动管理和更新Traefik反向代理中域名解析的工具。它支持通过DNS提供商（如AdGuard和Dnspod）来自动添加或更新CNAME记录。

## 环境变量配置
以下是需要配置的环境变量：

1. **TRAEFIK_HOST**: Traefik的URL地址支持httpBasic 认证 username:password@url。
2. **POLL_INTERVAL**: 拉取间隔，用于指定拉取的频率。
3. **DNS_NAME**: DNS提供者的名称，如AdGuard或Dnspod。
4. **DNS_ID**: DNS提供者的ID。
5. **DNS_SECRET**: DNS提供者的密钥。
6. **DNS_REFRESH**: 是否刷新DNS记录。
7. **DNS_RECORD_VALUE**: DNS记录的值，支持IPv4、IPv6或域名。
8. **AD_GUARD_HOST**: AdGuard的主机地址。



## 使用说明
1. 确保Traefik已经运行并配置好。
2. 编译并运行该项目，可以通过命令行参数或环境变量配置。
3. 项目会自动从Traefik获取域名信息，并在指定的DNS提供商处添加或更新CNAME记录。


