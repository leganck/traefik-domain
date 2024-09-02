package dns

import (
	"fmt"
	"github.com/gmichels/adguard-client-go"
	"github.com/leganck/docker-traefik-domain/config"
	"github.com/leganck/docker-traefik-domain/traefik"
	"log"
)

var scheme = "http"

type AdGuard struct {
	name    string
	dnsConf *config.Config
	client  *adguard.ADG
}

func (a *AdGuard) Init(dnsConf *config.Config) error {
	timeout := 10
	adGuardHost := config.GetAdGuardHost()
	if adGuardHost == "" {
		return fmt.Errorf("adGuardHostAddr is empty")

	}
	client, _ := adguard.NewClient(&adGuardHost, &dnsConf.ID, &dnsConf.Secret, &scheme, &timeout)
	a.client = client
	a.dnsConf = dnsConf
	return nil
}
func (p *AdGuard) AddOrUpdateCname(domain string, domains []*traefik.Domain) error {
	rewrites, err := p.client.GetAllRewrites()
	if err != nil {
		log.Printf("GetAllRewrites failed: %v", err)
	}
	rewritesMap := make(map[string]string)
	for _, re := range *rewrites {
		rewritesMap[re.Domain] = re.Answer
	}

	var addList []*traefik.Domain
	var updateList []*traefik.Domain

	for _, entry := range domains {
		if rewritesMap[entry.CustomDomain] != "" {
			if rewritesMap[entry.CustomDomain] != p.dnsConf.RecordValue {
				updateList = append(updateList, entry)
			}

		} else {
			addList = append(addList, entry)
		}
	}

	err = p.Add(domain, addList)
	if err != nil {
		return err
	}

	err = p.updateRecord(domain, updateList)
	if err != nil {
		return err
	}
	return nil
}

func (p *AdGuard) updateRecord(domain string, list []*traefik.Domain) error {
	if list == nil {
		log.Printf("%s: '%s' no record to update", p.name, domain)
		return nil
	}
	for _, d := range list {
		rewrite, err := p.client.UpdateRewrite(adguard.RewriteEntry{
			Domain: d.CustomDomain,
			Answer: p.dnsConf.RecordValue,
		})
		if err != nil {
			return fmt.Errorf("%s :%s update failed: %v", p.name, domain, err)
		}
		log.Printf(" %s :%s update success: %v", p.name, domain, rewrite.Domain)
	}
	return nil
}

func (p *AdGuard) Add(domain string, list []*traefik.Domain) error {
	if list == nil {
		log.Printf("%s: '%s' no record to add", p.name, domain)
		return nil
	}
	for _, d := range list {
		rewrite, err := p.client.CreateRewrite(adguard.RewriteEntry{
			Domain: d.CustomDomain,
			Answer: p.dnsConf.RecordValue,
		})
		if err != nil {
			return fmt.Errorf("%s :%s add failed: %v", p.name, domain, err)
		}
		log.Printf(" %s :%s add success: %v value", p.name, domain, rewrite.Domain)
	}
	return nil
}
