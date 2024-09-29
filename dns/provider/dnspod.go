package provider

import (
	"github.com/leganck/dnspod-go"
	"github.com/leganck/docker-traefik-domain/config"
	"github.com/leganck/docker-traefik-domain/dns/model"
	"github.com/leganck/docker-traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

type DnsPod struct {
	logger *log.Entry
	client *dnspod.Client
}

func (p *DnsPod) Init(dnsConf *config.Config, log *log.Entry) error {
	p.client = dnspod.NewClient(dnspod.CommonParams{LoginToken: dnsConf.ID + "," + dnsConf.Secret, Format: "json"})
	p.logger = log
	return nil
}

func (p *DnsPod) List(domain string) ([]*model.Record, error) {

	list, _, err := p.client.Records.List(dnspod.ListParams{
		RecordParam: &dnspod.RecordParam{Domain: domain},
	})

	records := make([]*model.Record, 0)
	for _, record := range list {
		records = append(records, &model.Record{
			Id:           record.ID,
			Name:         record.Name,
			Type:         record.Type,
			Value:        record.Value,
			MainDomain:   domain,
			CustomDomain: record.Name + "." + domain,
		})
	}
	return records, err
}

func (p *DnsPod) UpdateRecord(value string, updateList []*model.Record) error {
	if len(updateList) == 0 {
		p.logger.Debugln("no record to update")
		return nil
	}

	for _, record := range updateList {
		if record.Value != value {
			_, _, err := p.client.Records.Update("", record.MainDomain, record.Id, dnspod.Record{
				Name:  record.Name,
				Type:  record.Type,
				Value: record.Value,
				Line:  "默认",
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

func (p *DnsPod) AddRecord(value, recordType string, list []*traefik.Domain) error {
	if list == nil {
		p.logger.Debugf("no record to add")
		return nil
	}
	var errorList []*traefik.Domain
	for _, d := range list {
		create, _, err := p.client.Records.Create(d.MainDomain, "", dnspod.Record{
			Name:   d.SubDomain,
			Type:   recordType,
			Value:  value,
			TTL:    "600",
			Line:   "默认",
			Status: "enable",
		})
		if err != nil {
			p.logger.Errorf("add record %s %s error: %v", d.CustomDomain, value, err)
			errorList = append(errorList, d)
			continue
		}
		p.logger.Infof("add record %s %s success", d.CustomDomain, create.Value)
	}
	p.logger.Printf("all record add success")
	return nil
}
