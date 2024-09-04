package dns

import (
	"fmt"
	"github.com/leganck/docker-traefik-domain/config"
	"github.com/leganck/docker-traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
	"strings"
)

var logger *log.Entry

type DnsProvider interface {
	Init(dnsConf *config.Config) error

	AddOrUpdateCname(domain string, domains []*traefik.Domain) error
}

func NewDNSProvider(dnsConf *config.Config) (DnsProvider, error) {
	var dnsProvider DnsProvider
	switch strings.ToLower(dnsConf.Name) {
	case "dnspod":
		dnsProvider = &DnsPod{
			name: dnsConf.Name,
		}
	case "adguard":
		dnsProvider = &AdGuard{
			name: dnsConf.Name,
		}
	default:
		return nil, fmt.Errorf("dns provider %s not found", dnsConf.Name)
	}
	err := dnsProvider.Init(dnsConf)
	if err != nil {
		return nil, fmt.Errorf("init dns provider %s error: %s", dnsConf.Name, err)
	}

	return dnsProvider, nil
}
