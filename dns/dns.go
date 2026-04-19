package dns

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/dns/provider"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

type DnsProvider interface {
	Init(cfg *provider.ProviderConfig, log *log.Entry) error

	List(domain string) ([]*model.Record, error)

	AddRecord(value, recordType string, list []*traefik.Domain) error

	UpdateRecord(value string, updateList []*model.Record) error

	DeleteRecord(list []*model.Record) error
}

type Provider struct {
	logger       *log.Entry
	id           string
	name         string
	recordValue  string
	recordType   string
	provider     DnsProvider
	switchConfig *config.SwitchConfig
}

func NewDNSProvider(cfg *provider.ProviderConfig, switchConfig *config.SwitchConfig, logger *log.Entry) (*Provider, error) {
	providerType := strings.ToLower(cfg.Type)

	var dnsProvider DnsProvider
	switch providerType {
	case "dnspod":
		dnsProvider = &provider.DnsPod{}
	case "adguard":
		dnsProvider = &provider.AdGuard{}
	case "cloudflare":
		dnsProvider = &provider.Cloudflare{}
	case "openwrt":
		dnsProvider = &provider.OpenWRT{}
	default:
		return nil, fmt.Errorf("dns provider %s not found", providerType)
	}

	err := dnsProvider.Init(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("init dns provider %s error: %s", providerType, err)
	}

	recordType, finalValue := detectRecordType(cfg.RecordValue)

	return &Provider{
		logger:       logger,
		id:           cfg.ProviderID,
		name:         cfg.Name,
		recordValue:  finalValue,
		recordType:   recordType,
		provider:     dnsProvider,
		switchConfig: switchConfig,
	}, nil
}

func detectRecordType(value string) (string, string) {
	if value == "" {
		return "A", value
	}
	ipv4Regex := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	ipv6Regex := `^([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4})$`
	domainRegex := `^(?:(?:[a-zA-Z0-9-]{0,61}[A-Za-z0-9]\.)+)(?:[A-Za-z]{2,})$`

	if matchRegex(value, ipv4Regex) {
		return "A", value
	}
	if matchRegex(value, ipv6Regex) {
		return "AAAA", value
	}
	if matchRegex(value, domainRegex) {
		return "CNAME", value + "."
	}
	return "A", value
}

func matchRegex(s, pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(s)
}

func (p *Provider) AddOrUpdateCname(domain string, domains []*traefik.Domain) error {
	var filteredDomains []*traefik.Domain
	if p.switchConfig != nil {
		for _, d := range domains {
			if p.switchConfig.ShouldSync(d.CustomDomain, p.id) {
				filteredDomains = append(filteredDomains, d)
			} else {
				p.logger.Debugf("Skipping %s for provider %s (disabled)", d.CustomDomain, p.name)
			}
		}
	} else {
		filteredDomains = domains
	}

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
			if record.Value != p.recordValue {
				updateList = append(updateList, &model.Record{
					Id:         record.Id,
					Name:       record.Name,
					Value:      p.recordValue,
					Type:       record.Type,
					MainDomain: domain,
				})
			}
		} else {
			addList = append(addList, d)
		}
	}

	err = p.provider.AddRecord(p.recordValue, p.recordType, addList)
	if err != nil {
		return err
	}
	err = p.provider.UpdateRecord(p.recordValue, updateList)
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) DeleteDomain(customDomain string) error {
	_, mainDomain, err := model.SplitDomain(customDomain)
	if err != nil {
		return fmt.Errorf("failed to parse domain %s: %w", customDomain, err)
	}

	records, err := p.provider.List(mainDomain)
	if err != nil {
		return fmt.Errorf("failed to list records for %s: %w", mainDomain, err)
	}

	var toDelete []*model.Record
	for _, r := range records {
		if strings.EqualFold(r.CustomDomain, customDomain) {
			toDelete = append(toDelete, r)
		}
	}

	if len(toDelete) == 0 {
		p.logger.Debugf("No records found to delete for %s", customDomain)
		return nil
	}

	return p.provider.DeleteRecord(toDelete)
}

func (p *Provider) ListRecords(domain string) ([]*model.Record, error) {
	return p.provider.List(domain)
}
