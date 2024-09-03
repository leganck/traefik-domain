package util

import "strings"

func ParseUrl(url string) (string, string, string) {
	authInfo, remainingURL := separateAuthInfo(url)
	username, password := parseAuth(authInfo)

	return remainingURL, username, password
}

// 分离认证信息（用户名和密码）和剩余的URL
func separateAuthInfo(url string) (string, string) {
	atIndex := strings.Index(url, "@")
	if atIndex == -1 {
		return "", url
	}
	return url[:atIndex], url[atIndex+1:]
}

// 解析认证信息为用户名和密码
func parseAuth(authInfo string) (string, string) {
	colonIndex := strings.Index(authInfo, ":")
	if colonIndex == -1 {
		return "", ""
	}
	return authInfo[:colonIndex], authInfo[colonIndex+1:]
}

// 解析主机名和端口
func parseHostAndPort(url string) (string, string) {
	protocolEndIndex := strings.Index(url, "://")
	if protocolEndIndex == -1 {
		return "", ""
	}

	hostAndPort := url[protocolEndIndex+3:]

	colonIndex := strings.Index(hostAndPort, ":")
	if colonIndex == -1 {
		return hostAndPort, ""
	}

	return hostAndPort[:colonIndex], hostAndPort[colonIndex+1:]
}
