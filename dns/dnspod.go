package dns

import (
	"github.com/leganck/dnspod-go"
	"github.com/leganck/docker-traefik-domain/config"
	"github.com/leganck/docker-traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

type DnsPod struct {
	name    string
	dnsConf *config.Config
	client  *dnspod.Client
}

func (p *DnsPod) Init(dnsConf *config.Config) error {
	p.dnsConf = dnsConf
	p.name = "DnsPod"
	p.client = dnspod.NewClient(dnspod.CommonParams{LoginToken: dnsConf.ID + "," + dnsConf.Secret, Format: "json"})
	logger = log.WithField("provider", p.name)

	return nil
}

func (p *DnsPod) list(domain string) ([]dnspod.Record, error) {

	list, _, err := p.client.Records.List(dnspod.ListParams{
		RecordParam: &dnspod.RecordParam{Domain: domain},
		RecordType:  p.dnsConf.RecordType,
	})
	return list, err
}

func (p *DnsPod) AddOrUpdateCname(domain string, domains []*traefik.Domain) error {
	logger = logger.WithField("domain", domain)
	domainMap := make(map[string]dnspod.Record)

	list, err := p.list(domain)
	for _, d := range list {
		domainMap[d.Name] = d
	}

	if err != nil {
		logger.Warningf("'%s' list error: %v", domain, err)
		return err
	}
	var updateList = make(map[dnspod.Record]*traefik.Domain)
	var addList []*traefik.Domain

	for _, d := range domains {
		record, ok := domainMap[d.SubDomain]
		if ok {
			if record.Value != p.dnsConf.RecordValue {
				updateList[domainMap[d.SubDomain]] = d
			}
		} else {
			addList = append(addList, d)
		}
	}

	err = p.AddRecord(domain, addList)
	if err != nil {
		return err
	}
	err = p.updateCname(domain, updateList)
	if err != nil {
		return err
	}
	return nil

}

func (p *DnsPod) updateCname(domain string, updateList map[dnspod.Record]*traefik.Domain) error {
	if len(updateList) == 0 {
		logger.Debugln("no record to update")
		return nil
	}
	var errorList []*traefik.Domain

	if p.dnsConf.Refresh {
		for record, v := range updateList {
			if record.Value != p.dnsConf.RecordValue {
				record.Value = p.dnsConf.RecordValue
				record.Type = p.dnsConf.RecordType
				_, _, err := p.client.Records.Update("", domain, record.ID, record)
				if err != nil {
					logger.Errorf("update record %s %s error: %v", record.Name, p.dnsConf.RecordValue, err)
					errorList = append(errorList, v)
					continue
				}
			} else {
				logger.Infof("record %s %s no need update", record.Name, record.Value)
			}
		}
	}
	logger.Infof("all record update success")
	return nil
}

func (p *DnsPod) AddRecord(domain string, list []*traefik.Domain) error {
	if list == nil {
		logger.Debugf("no record to add")
		return nil
	}
	var errorList []*traefik.Domain
	for _, d := range list {
		create, _, err := p.client.Records.Create(domain, "", dnspod.Record{
			Name:   d.SubDomain,
			Type:   p.dnsConf.RecordType,
			Value:  p.dnsConf.RecordValue,
			TTL:    "600",
			Line:   "默认",
			Status: "enable",
		})
		if err != nil {
			logger.Errorf("add record %s %s error: %v", d.SubDomain, p.dnsConf.RecordValue, err)
			errorList = append(errorList, d)
			continue
		}
		logger.Infof("add record %s %s success", create.Name, create.Value)
	}
	logger.Printf("all record add success")
	return nil
}
