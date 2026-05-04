package provider

import (
	"github.com/leganck/dnspod-go"
	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns/internal/providerutil"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

type DnsPod struct {
	logger *log.Entry
	client *dnspod.Client
}

func (p *DnsPod) Init(cfg *config.ProviderConfig, log *log.Entry) error {
	p.client = dnspod.NewClient(dnspod.CommonParams{LoginToken: cfg.ID + "," + cfg.Secret, Format: "json"})
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
			Managed:      record.Remark == providerutil.RecordRemark,
		})
	}
	return records, err
}

func (p *DnsPod) UpdateRecord(value string, updateList []*model.Record, overwrite bool) error {
	return providerutil.UpdateRecords(p.logger, value, updateList, overwrite, func(record *model.Record) error {
		_, _, err := p.client.Records.Update("", record.MainDomain, record.Id, dnspod.Record{
			Name:  record.Name,
			Type:  record.Type,
			Value: value,
			Line:  "默认",
		})
		return err
	})
}

func (p *DnsPod) AddRecord(value, recordType string, list []*traefik.Domain) error {
	return providerutil.AddDomains(p.logger, value, list, func(d *traefik.Domain) (string, error) {
		create, _, err := p.client.Records.Create(d.MainDomain, "", dnspod.Record{
			Name:   d.SubDomain,
			Type:   recordType,
			Value:  value,
			TTL:    "600",
			Line:   "默认",
			Status: "enable",
			Remark: providerutil.RecordRemark,
		})
		if err != nil {
			return "", err
		}

		return create.Value, nil
	})
}

func (p *DnsPod) DeleteRecord(list []*model.Record) error {
	return providerutil.DeleteManagedRecords(p.logger, list, func(record *model.Record) error {
		_, err := p.client.Records.Delete(0, record.MainDomain, record.Id)
		return err
	})
}
