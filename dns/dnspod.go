package dns

import (
	"github.com/leganck/dnspod-go"
	"github.com/leganck/docker-traefik-domain/config"
	"github.com/leganck/docker-traefik-domain/traefik"
	"log"
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

	domainMap := make(map[string]dnspod.Record)

	list, err := p.list(domain)
	for _, d := range list {
		domainMap[d.Name] = d
	}

	if err != nil {
		log.Printf("%s: '%s' list error: %v\n", p.name, domain, err)
		return err
	}
	var updateList = make(map[dnspod.Record]*traefik.Domain)
	var addList []*traefik.Domain

	for _, d := range domains {
		_, ok := domainMap[d.SubDomain]
		if ok {
			if d.CustomDomain == p.dnsConf.RecordValue {
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
		log.Printf("%s: '%s' no record to update\n", p.name, domain)
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
					log.Printf("%s update record %s %s error: %v\n", domain, record.Name, p.dnsConf.RecordValue, err)
					errorList = append(errorList, v)
					continue
				}
			} else {
				log.Printf("%s record %s %s no need update\n", domain, record.Name, record.Value)
			}
		}
	}
	log.Printf("%s:all record update success\n", p.name)
	return nil
}

func (p *DnsPod) AddRecord(domain string, list []*traefik.Domain) error {
	if list == nil {
		log.Printf("%s: '%s' no record to add", p.name, domain)
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
			log.Printf("%s:%s add record %s %s error: %v\n", p.name, domain, d.SubDomain, p.dnsConf.RecordValue, err)
			errorList = append(errorList, d)
			continue
		}
		log.Printf("%s:%s add record %s %s success\n", p.name, domain, create.Name, create.Value)
	}
	log.Printf("%s:all record add success\n", p.name)
	return nil
}
