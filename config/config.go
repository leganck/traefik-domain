package config

import (
	"fmt"
	"regexp"
	"sort"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	v         *viper.Viper
	logConfig = map[string]int{
		"time":     0,
		"level":    1,
		"provider": 2,
		"domain":   3,
		"msg":      4,
	}
	ipv4Regex   = regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	ipv6Regex   = regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4})$`)
	domainRegex = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9-]{0,61}[A-Za-z0-9]\.)+)(?:[A-Za-z]{2,})$`)
)

type Config struct {
	Name         string
	ID           string
	Secret       string
	Refresh      bool
	RecordValue  string
	RecordType   string
	PollInterval int
	LogLevel     log.Level
	TraefikHost  string
	AdGuardHost  string
}

func init() {
	v = viper.New()
	v.SetEnvPrefix("")
	v.AutomaticEnv()

	v.SetDefault("poll-interval", 5)
	v.SetDefault("log-level", "info")
	v.SetDefault("dns-refresh", false)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.AddConfigPath("/etc/traefik-domain/")
	v.AddConfigPath("$HOME/.traefik-domain/")

	if err := v.ReadInConfig(); err == nil {
		log.Infof("Using config file: %s", v.ConfigFileUsed())
	} else {
		log.Printf("No config file found, using env vars: %v", err)
	}

	v.RegisterAlias("TRAEFIK_HOST", "traefik.host")
	v.RegisterAlias("POLL_INTERVAL", "poll.interval")
	v.RegisterAlias("DNS_NAME", "dns.name")
	v.RegisterAlias("DNS_ID", "dns.id")
	v.RegisterAlias("DNS_SECRET", "dns.secret")
	v.RegisterAlias("DNS_REFRESH", "dns.refresh")
	v.RegisterAlias("DNS_RECORD_VALUE", "dns.record.value")
	v.RegisterAlias("AD_GUARD_HOST", "adguard.host")
	v.RegisterAlias("LOG_LEVEL", "log.level")
}

func GetConfig() (*Config, error) {
	levelStr := v.GetString("log-level")
	level, err := log.ParseLevel(levelStr)
	if err != nil {
		log.Errorf("invalid log level %s", levelStr)
		level = log.InfoLevel
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		SortingFunc: func(keys []string) {
			sort.Slice(keys, func(i, j int) bool {
				vi, ok1 := logConfig[keys[i]]
				vj, ok2 := logConfig[keys[j]]
				if !ok1 {
					vi = 99
				}
				if !ok2 {
					vj = 99
				}
				return vi < vj
			})
		},
	})
	log.SetLevel(level)

	dnsName := v.GetString("dns.name")
	dnsID := v.GetString("dns.id")
	dnsSecret := v.GetString("dns.secret")
	recordValue := v.GetString("dns.record.value")

	if dnsName == "" || dnsID == "" || dnsSecret == "" {
		return nil, fmt.Errorf("invalid dns config: name=%s, id=%s, secret=%s", dnsName, dnsID, dnsSecret)
	}
	if recordValue == "" {
		return nil, fmt.Errorf("invalid domain value")
	}

	recordType, finalValue := detectRecordType(recordValue)

	pollInterval := v.GetInt("poll-interval")
	if pollInterval <= 0 {
		pollInterval = 5
	}

	return &Config{
		Name:         dnsName,
		ID:           dnsID,
		Secret:       dnsSecret,
		Refresh:      v.GetBool("dns.refresh"),
		RecordValue:  finalValue,
		RecordType:   recordType,
		PollInterval: pollInterval,
		LogLevel:     level,
		TraefikHost:  v.GetString("traefik.host"),
		AdGuardHost:  v.GetString("adguard.host"),
	}, nil
}

func detectRecordType(value string) (string, string) {
	if isIPv4(value) {
		return "A", value
	}
	if isIPv6(value) {
		return "AAAA", value
	}
	if isDomain(value) {
		return "CNAME", value + "."
	}
	return "A", value
}

func isIPv4(s string) bool {
	return ipv4Regex.MatchString(s)
}

func isIPv6(s string) bool {
	return ipv6Regex.MatchString(s)
}

func isDomain(s string) bool {
	return domainRegex.MatchString(s)
}

func GetTraefikHost() string {
	return v.GetString("traefik.host")
}

func GetAdGuardHost() string {
	return v.GetString("adguard.host")
}

func ReloadConfig() error {
	return v.ReadInConfig()
}
