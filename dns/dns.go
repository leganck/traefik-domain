package dns

import (
	"fmt"
	"strings"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/dns/provider"
	"github.com/leganck/traefik-domain/traefik"
	"github.com/leganck/traefik-domain/web"
	log "github.com/sirupsen/logrus"
)

type DnsProvider interface {
	Init(dnsConf *config.Config, log *log.Entry) error

	List(domain string) ([]*model.Record, error)

	AddRecord(value, recordType string, list []*traefik.Domain) error

	UpdateRecord(value string, updateList []*model.Record) error
}

type Provider struct {
	logger       *log.Entry
	name         string
	dnsConf      *config.Config
	provider     DnsProvider
	switchConfig *web.SwitchConfig
}

func NewDNSProvider(dnsConf *config.Config, switchConfig *web.SwitchConfig) (*Provider, error) {
	providerName := strings.ToLower(dnsConf.Name)

	logger := log.WithFields(log.Fields{"provider": providerName})

	var dnsProvider DnsProvider
	switch providerName {
	case "dnspod":
		dnsProvider = &provider.DnsPod{}
	case "adguard":
		dnsProvider = &provider.AdGuard{}
	case "cloudflare":
		dnsProvider = &provider.Cloudflare{}
	case "openwrt":
		dnsProvider = &provider.OpenWRT{}
	default:
		return nil, fmt.Errorf("dns provider %s not found", providerName)
	}
	err := dnsProvider.Init(dnsConf, logger)
	if err != nil {
		return nil, fmt.Errorf("init dns provider %s error: %s", providerName, err)
	}
	return &Provider{
		logger:       logger,
		dnsConf:      dnsConf,
		name:         providerName,
		provider:     dnsProvider,
		switchConfig: switchConfig,
	}, nil
}

func (p *Provider) AddOrUpdateCname(domain string, domains []*traefik.Domain) error {
	// Filter domains based on switch config
	var filteredDomains []*traefik.Domain
	if p.switchConfig != nil {
		for _, d := range domains {
			if p.switchConfig.ShouldSync(d.CustomDomain, p.name) {
				filteredDomains = append(filteredDomains, d)
			} else {
				p.logger.Debugf("Skipping %s for provider %s (disabled)", d.CustomDomain, p.name)
			}
		}
	} else {
		// If no switch config, sync all domains
		filteredDomains = domains
	}

	// If no domains to sync, return early
	if len(filteredDomains) == 0 {
		p.logger.Debugf("No domains to sync for provider %s", p.name)
		return nil
	}

	domainMap := make(map[string]*model.Record)

	list, err := p.provider.List(domain)
	if err != nil {
		p.logger.Warningf("'%s' List error: %v", domain, err)
		return err
	}

	for _, d := range list {
		domainMap[d.Name] = d
	}

	var updateList = make([]*model.Record, 0)
	var addList []*traefik.Domain

	for _, d := range filteredDomains {
		record, ok := domainMap[d.SubDomain]
		if ok {
			if record.Value != p.dnsConf.RecordValue {
				updateList = append(updateList, &model.Record{
					Id:         record.Id,
					Name:       record.Name,
					Value:      p.dnsConf.RecordValue,
					Type:       record.Type,
					MainDomain: domain,
				})
			}
		} else {
			addList = append(addList, d)
		}
	}

	err = p.provider.AddRecord(p.dnsConf.RecordValue, p.dnsConf.RecordType, addList)
	if err != nil {
		return err
	}
	err = p.provider.UpdateRecord(domain, updateList)
	if err != nil {
		return err
	}
	return nil
}
