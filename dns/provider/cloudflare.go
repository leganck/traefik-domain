package provider

import (
	"fmt"
	"sync"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns/internal/providerutil"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type Cloudflare struct {
	logger     *log.Entry
	client     *cf.API
	background context.Context
	zoneCache  map[string]*cf.ResourceContainer
	zoneMutex  sync.RWMutex
}

func (p *Cloudflare) Init(cfg *config.ProviderConfig, log *log.Entry) error {
	apiClient, err := cf.NewWithAPIToken(cfg.Secret)
	if err != nil {
		log.Errorf("init cloudflare client error: %v", err)
		return fmt.Errorf("init cloudflare client error: %v", err)
	}
	p.client = apiClient
	p.logger = log
	p.background = context.Background()
	p.zoneCache = make(map[string]*cf.ResourceContainer)
	return nil
}

func (p *Cloudflare) List(domain string) ([]*model.Record, error) {
	zoneIdentifier, err := p.zoneIdentifier(domain)
	if err != nil {
		return nil, err
	}
	list, _, err := p.client.ListDNSRecords(p.background, zoneIdentifier, cf.ListDNSRecordsParams{})

	if err != nil {
		p.logger.Errorf("list dns record error: %v", err)
		return nil, fmt.Errorf("list dns record error: %v", err)
	}

	records := make([]*model.Record, 0)
	for _, record := range list {
		subDomain, mainDomain, err := model.SplitDomain(record.Name)
		if err != nil {
			p.logger.Errorf("parse domain : %s  failed : %v", record.Name, err)
			continue
		}

		records = append(records, &model.Record{
			Id:           record.ID,
			Name:         subDomain,
			Type:         record.Type,
			Value:        record.Content,
			MainDomain:   mainDomain,
			CustomDomain: record.Name,
			Managed:      record.Comment == providerutil.RecordRemark,
		})
	}
	return records, err
}

func (p *Cloudflare) UpdateRecord(value string, updateList []*model.Record, overwrite bool) error {
	return providerutil.UpdateRecords(p.logger, value, updateList, overwrite, func(record *model.Record) error {
		identifier, err := p.zoneIdentifier(record.MainDomain)
		if err != nil {
			p.logger.Errorf("get zone identifier error: %v", err)
			return err
		}

		_, err = p.client.UpdateDNSRecord(p.background, identifier, cf.UpdateDNSRecordParams{
			ID:      record.Id,
			Name:    record.Name,
			Type:    record.Type,
			Content: value,
		})
		return err
	})
}

func (p *Cloudflare) AddRecord(value, recordType string, list []*traefik.Domain) error {
	return providerutil.AddDomains(p.logger, value, list, func(d *traefik.Domain) (string, error) {
		identifier, err := p.zoneIdentifier(d.MainDomain)
		if err != nil {
			p.logger.Errorf("get zone identifier error: %v", err)
			return "", err
		}

		_, err = p.client.CreateDNSRecord(p.background, identifier, cf.CreateDNSRecordParams{
			Name:    d.SubDomain,
			Content: value,
			Type:    recordType,
			Comment: providerutil.RecordRemark,
		})
		if err != nil {
			return "", err
		}

		return value, nil
	})
}

func (p *Cloudflare) DeleteRecord(list []*model.Record) error {
	return providerutil.DeleteManagedRecords(p.logger, list, func(record *model.Record) error {
		identifier, err := p.zoneIdentifier(record.MainDomain)
		if err != nil {
			p.logger.Errorf("get zone identifier error: %v", err)
			return err
		}

		return p.client.DeleteDNSRecord(p.background, identifier, record.Id)
	})
}

func (p *Cloudflare) zoneIdentifier(domain string) (*cf.ResourceContainer, error) {
	p.zoneMutex.RLock()
	zone, exists := p.zoneCache[domain]
	p.zoneMutex.RUnlock()

	if exists {
		return zone, nil
	}

	zones, err := p.client.ListZones(p.background, domain)
	if err != nil {
		p.logger.Errorf("list zone error: %v", err)
		return nil, fmt.Errorf("list zone error: %v", err)
	}

	if len(zones) == 0 {
		p.logger.Errorf("no zone found for domain %s", domain)
		return nil, fmt.Errorf("no zone found for domain %s", domain)
	}

	zoneIdentifier := cf.ZoneIdentifier(zones[0].ID)
	p.zoneMutex.Lock()
	p.zoneCache[domain] = zoneIdentifier
	p.zoneMutex.Unlock()
	return zoneIdentifier, err
}
