package config

import (
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"sort"
	"strconv"
)

var confPath = flag.String("conf", os.Getenv("CONF"), "config path")

var logLevel = flag.String("log-level", os.Getenv("LOG_LEVEL"), "config path")

var traefikHost = flag.String("traefik-host", os.Getenv("TRAEFIK_HOST"), "traefik url")

var pollInterval = flag.String("poll-interval", os.Getenv("POLL_INTERVAL"), "poll interval")

var dnsName = flag.String("dns-name", os.Getenv("DNS_NAME"), "dns name dnspod adGuard")

var dnsId = flag.String("dns-id", os.Getenv("DNS_ID"), "dns provider id")

var dnsSecret = flag.String("dns-secret", os.Getenv("DNS_SECRET"), "dns provider secret")

var dnsRefresh = flag.Bool("dns-refresh", os.Getenv("DNS_REFRESH") == "true", " enable refresh dns record")

var dnsRecordValue = flag.String("dns-record-value", os.Getenv("DNS_RECORD_VALUE"), "dns record value support ipv4 ipv6 domain")

var adGuardHost = flag.String("ad-guard-host", os.Getenv("AD_GUARD_HOST"), "adGuardHostAddr")

var fileConfig = make(map[string]string)

var logConfig = map[string]int{
	"time":     0,
	"level":    1,
	"provider": 2,
	"domain":   3,
	"msg":      4,
}

type Config struct {
	Name         string
	ID           string
	Secret       string
	Refresh      bool
	RecordValue  string
	RecordType   string
	PollInterval int
	LogLevel     log.Level
}

func init() {
	flag.Parse()
	envMap, err := godotenv.Read(*confPath)
	if err != nil {
		log.Printf("No .env file found, using environment variables")
	} else {
		fileConfig = envMap
		if *traefikHost == "" {
			*traefikHost = fileConfig["TRAEFIK_HOST"]
		}
		if *pollInterval == "" {

			value, exists := fileConfig["POLL_INTERVAL"]
			if !exists {
				value = "5"
			}
			*pollInterval = value
		}
		if *dnsName == "" {
			*dnsName = fileConfig["DNS_NAME"]
		}
		if *dnsId == "" {
			*dnsId = fileConfig["DNS_ID"]
		}
		if *dnsSecret == "" {
			*dnsSecret = fileConfig["DNS_SECRET"]
		}

		if !*dnsRefresh {
			*dnsRefresh = fileConfig["DNS_REFRESH"] == "true"
		}
		if *dnsRecordValue == "" {
			*dnsRecordValue = fileConfig["DNS_RECORD_VALUE"]
		}
		if *adGuardHost == "" {
			*adGuardHost = fileConfig["AD_GUARD_HOST"]
		}
		if *logLevel == "" {
			s, ok := fileConfig["LOG_LEVEL"]
			if ok {
				*logLevel = s
			} else {
				*logLevel = "info"
			}
		}
	}

}

func GetConfig() (*Config, error) {
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("invalid log level %s", *logLevel)
	}
	log.SetFormatter(&log.TextFormatter{
		SortingFunc: func(keys []string) {
			sort.Slice(keys, func(i, j int) bool {
				return logConfig[keys[i]] < logConfig[keys[j]]
			})
		},
	})
	log.SetLevel(level)

	if *dnsName == "" || *dnsId == "" || *dnsSecret == "" {
		return nil, fmt.Errorf("invalid dns config")
	}
	if *dnsRecordValue == "" {
		return nil, fmt.Errorf("invalid  domain value")
	}

	RecordType := "A"
	RecordValue := *dnsRecordValue
	switch {
	case isIPv4(RecordValue):
		RecordType = "A"
		break
	case isIPv6(RecordValue):
		RecordType = "AAAA"
		break
	case isDomain(RecordValue):
		RecordType = "CNAME"
		RecordValue = RecordValue + "."
		break
	default:
		return nil, fmt.Errorf("invalid domain value")
	}

	pollInterval, err := strconv.Atoi(*pollInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid poll interval")
	}

	return &Config{
		Name:         *dnsName,
		ID:           *dnsId,
		Secret:       *dnsSecret,
		RecordValue:  RecordValue,
		RecordType:   RecordType,
		Refresh:      *dnsRefresh,
		PollInterval: pollInterval,
		LogLevel:     level,
	}, nil
}

// isIPv4 检查字符串是否为有效的IPv4地址
func isIPv4(s string) bool {
	ipv4Regex := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	matched, _ := regexp.MatchString(ipv4Regex, s)
	return matched
}

// isIPv6 检查字符串是否为有效的IPv6地址
func isIPv6(s string) bool {
	ipv6Regex := `^([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4})$`
	matched, _ := regexp.MatchString(ipv6Regex, s)
	return matched
}

// isDomain 检查字符串是否为有效的域名
func isDomain(s string) bool {
	domainRegex := `^(?:(?:[a-zA-Z0-9-]{0,61}[A-Za-z0-9]\.)+)(?:[A-Za-z]{2,})$`
	matched, _ := regexp.MatchString(domainRegex, s)
	return matched
}

func GetTraefikHost() string {
	return *traefikHost
}

func GetAdGuardHost() string {
	return *adGuardHost
}
