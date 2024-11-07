package provider

import (
	"fmt"
	"github.com/gmichels/adguard-client-go"
	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

var scheme = "http"

type AdGuard struct {
	logger *log.Entry
	client *adguard.ADG
}

func (a *AdGuard) Init(dnsConf *config.Config, log *log.Entry) error {
	timeout := 10
	adGuardHost := config.GetAdGuardHost()
	if adGuardHost == "" {
		return fmt.Errorf("adGuardHostAddr is empty")

	}
	client, _ := adguard.NewClient(&adGuardHost, &dnsConf.ID, &dnsConf.Secret, &scheme, &timeout)
	a.client = client
	a.logger = log
	return nil
}

func (p *AdGuard) List(domain string) ([]*model.Record, error) {
	rewrites, err := p.client.GetAllRewrites()
	if err != nil {
		p.logger.Errorf("GetAllRewrites failed: %v", err)
		return nil, fmt.Errorf("GetAllRewrites failed: %v", err)
	}
	result := make([]*model.Record, 0)
	for _, re := range *rewrites {
		subDomain, mainDomain, err := model.SplitDomain(re.Domain)
		if err != nil {
			p.logger.Errorf("parse domain : %s  failed : %v", re.Domain, err)
			continue
		}
		if mainDomain == domain {
			result = append(result, &model.Record{
				Name:         subDomain,
				MainDomain:   mainDomain,
				CustomDomain: re.Domain,
				Value:        re.Answer,
			})
		}
	}
	return result, nil
}

func (p *AdGuard) UpdateRecord(value string, list []*model.Record) error {
	if list == nil {
		p.logger.Debugln("no record to update")
		return nil
	}
	for _, d := range list {
		rewrite, err := p.client.UpdateRewrite(adguard.RewriteEntry{
			Domain: d.CustomDomain,
			Answer: value,
		})
		if err != nil {
			p.logger.Errorf("update failed: %v", d.CustomDomain)
			continue
		}
		p.logger.Infof("update success: %v", rewrite.Domain)
	}
	return nil
}

func (p *AdGuard) AddRecord(value, _ string, list []*traefik.Domain) error {
	if list == nil {
		p.logger.Debugf("no record to add")
		return nil
	}
	for _, d := range list {
		rewrite, err := p.client.CreateRewrite(adguard.RewriteEntry{
			Domain: d.CustomDomain,
			Answer: value,
		})
		if err != nil {
			return fmt.Errorf("%s add failed: %v", d.CustomDomain, err)
		}
		p.logger.Printf(":%s add success: %v value", d.CustomDomain, rewrite.Domain)
	}
	return nil
}
