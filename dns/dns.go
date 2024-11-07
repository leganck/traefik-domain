package dns

import (
	"fmt"
	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/dns/provider"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
	"strings"
)

type DnsProvider interface {
	Init(dnsConf *config.Config, log *log.Entry) error

	List(domain string) ([]*model.Record, error)

	AddRecord(value, recordType string, list []*traefik.Domain) error

	UpdateRecord(value string, updateList []*model.Record) error
}

type Provider struct {
	logger   *log.Entry
	name     string
	dnsConf  *config.Config
	provider DnsProvider
}

func NewDNSProvider(dnsConf *config.Config) (*Provider, error) {
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
	default:
		return nil, fmt.Errorf("dns provider %s not found", providerName)
	}
	err := dnsProvider.Init(dnsConf, logger)
	if err != nil {
		return nil, fmt.Errorf("init dns provider %s error: %s", providerName, err)
	}
	return &Provider{
		logger:   logger,
		dnsConf:  dnsConf,
		name:     providerName,
		provider: dnsProvider,
	}, nil
}

func (p *Provider) AddOrUpdateCname(domain string, domains []*traefik.Domain) error {
	domainMap := make(map[string]*model.Record)

	list, err := p.provider.List(domain)
	for _, d := range list {
		domainMap[d.Name] = d
	}

	if err != nil {
		p.logger.Warningf("'%s' List error: %v", domain, err)
		return err
	}
	var updateList = make([]*model.Record, 0)
	var addList []*traefik.Domain

	for _, d := range domains {
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
