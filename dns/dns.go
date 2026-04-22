package dns

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/dns/provider"
	"github.com/leganck/traefik-domain/internal/state"
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
	switchConfig *state.DomainSyncState
}

var (
	ipv4Regex   = regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	ipv6Regex   = regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4})$`)
	domainRegex = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9-]{0,61}[A-Za-z0-9]\.)+)(?:[A-Za-z]{2,})$`)
)

func NewDNSProvider(cfg *provider.ProviderConfig, switchConfig *state.DomainSyncState, logger *log.Entry) (*Provider, error) {
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
	if ipv4Regex.MatchString(value) {
		return "A", value
	}
	if ipv6Regex.MatchString(value) {
		return "AAAA", value
	}
	if domainRegex.MatchString(value) {
		return "CNAME", value + "."
	}
	return "A", value
}

func (p *Provider) EnsureDomain(customDomain string, overwrite bool) error {
	subDomain, mainDomain, err := model.SplitDomain(customDomain)
	if err != nil {
		return fmt.Errorf("failed to parse domain %s: %w", customDomain, err)
	}

	records, err := p.provider.List(mainDomain)
	if err != nil {
		return fmt.Errorf("failed to list records for %s: %w", mainDomain, err)
	}

	var existing *model.Record
	for _, r := range records {
		if strings.EqualFold(r.CustomDomain, customDomain) {
			existing = r
			break
		}
	}

	if existing == nil {
		return p.provider.AddRecord(p.recordValue, p.recordType, []*traefik.Domain{{
			MainDomain:   mainDomain,
			SubDomain:    subDomain,
			CustomDomain: customDomain,
		}})
	}

	if !existing.Managed && !overwrite {
		return fmt.Errorf("record %s exists but is not managed by traefik-domain", customDomain)
	}

	if existing.Value == p.recordValue && existing.Type == p.recordType && existing.Managed {
		p.logger.Debugf("record %s already matches desired state", customDomain)
		return nil
	}

	return p.provider.UpdateRecord(p.recordValue, []*model.Record{{
		Id:           existing.Id,
		Name:         existing.Name,
		Value:        p.recordValue,
		Type:         existing.Type,
		MainDomain:   mainDomain,
		CustomDomain: customDomain,
		Managed:      true,
	}})
}

func (p *Provider) DeleteManagedDomain(customDomain string) error {
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
		if strings.EqualFold(r.CustomDomain, customDomain) && r.Managed {
			toDelete = append(toDelete, r)
		}
	}

	if len(toDelete) == 0 {
		p.logger.Debugf("No managed records found to delete for %s", customDomain)
		return nil
	}

	return p.provider.DeleteRecord(toDelete)
}

func (p *Provider) ListRecords(domain string) ([]*model.Record, error) {
	return p.provider.List(domain)
}
