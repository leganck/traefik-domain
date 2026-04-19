package provider

import (
	"fmt"
	"sync"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type Cloudflare struct {
	logger     *log.Entry
	client     *cf.API
	background context.Context
}

var (
	domainZone = map[string]*cf.ResourceContainer{}
	zoneMutex  sync.RWMutex
)

func (p *Cloudflare) Init(cfg *ProviderConfig, log *log.Entry) error {
	apiClient, err := cf.NewWithAPIToken(cfg.Secret)
	if err != nil {
		log.Errorf("init cloudflare client error: %v", err)
		return fmt.Errorf("init cloudflare client error: %v", err)
	}
	p.client = apiClient
	p.logger = log
	p.background = context.Background()
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
		})
	}
	return records, err
}

func (p *Cloudflare) UpdateRecord(value string, updateList []*model.Record) error {

	if len(updateList) == 0 {
		p.logger.Debugln("no record to update")
		return nil
	}
	for _, record := range updateList {
		if record.Value != value {
			identifier, err := p.zoneIdentifier(record.MainDomain)
			if err != nil {
				p.logger.Errorf("get zone identifier error: %v", err)
				continue
			}

			_, err = p.client.UpdateDNSRecord(p.background, identifier, cf.UpdateDNSRecordParams{
				ID:      record.Id,
				Name:    record.Name,
				Type:    record.Type,
				Content: value,
			})

			if err != nil {
				p.logger.Errorf("update record %s %s error: %v", record.CustomDomain, value, err)
				continue
			}
		} else {
			p.logger.Infof("record %s %s no need update", record.CustomDomain, record.Value)
		}
	}
	p.logger.Infof("all record update success")
	return nil
}

func (p *Cloudflare) AddRecord(value, recordType string, list []*traefik.Domain) error {
	if list == nil {
		p.logger.Debugf("no record to add")
		return nil
	}
	for _, d := range list {
		identifier, err := p.zoneIdentifier(d.MainDomain)
		if err != nil {
			p.logger.Errorf("get zone identifier error: %v", err)
			continue
		}

		_, err = p.client.CreateDNSRecord(p.background, identifier, cf.CreateDNSRecordParams{
			Name:    d.SubDomain,
			Content: value,
			Type:    recordType,
		})

		if err != nil {
			p.logger.Errorf("add record %s %s error: %v", d.CustomDomain, value, err)
			continue
		}
		p.logger.Infof("add record %s %s success", d.CustomDomain, value)
	}
	p.logger.Printf("all record add success")
	return nil
}

func (p *Cloudflare) DeleteRecord(list []*model.Record) error {
	if len(list) == 0 {
		p.logger.Debugln("no record to delete")
		return nil
	}
	for _, record := range list {
		identifier, err := p.zoneIdentifier(record.MainDomain)
		if err != nil {
			p.logger.Errorf("get zone identifier error: %v", err)
			continue
		}
		if err := p.client.DeleteDNSRecord(p.background, identifier, record.Id); err != nil {
			p.logger.Errorf("delete record %s error: %v", record.CustomDomain, err)
			continue
		}
		p.logger.Infof("delete record %s success", record.CustomDomain)
	}
	return nil
}

func (p *Cloudflare) zoneIdentifier(domain string) (*cf.ResourceContainer, error) {
	zoneMutex.RLock()
	zone, exists := domainZone[domain]
	zoneMutex.RUnlock()

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
	zoneMutex.Lock()
	domainZone[domain] = zoneIdentifier
	zoneMutex.Unlock()
	return zoneIdentifier, err
}
