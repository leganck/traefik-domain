package traefik

import (
	"encoding/json"
	"fmt"
	"github.com/leganck/docker-traefik-domain/config"
	"github.com/leganck/docker-traefik-domain/util"
	"golang.org/x/net/publicsuffix"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Domain struct {
	Domain       string `json:"domain"`
	SubDomain    string `json:"sub"`
	CustomDomain string `json:"customDomain"`
}

func (d *Domain) String() string {
	return d.CustomDomain
}

type RouterInfo struct {
	EntryPoints []string `json:"entryPoints,omitempty" toml:"entryPoints,omitempty" yaml:"entryPoints,omitempty" export:"true"`
	Middlewares []string `json:"middlewares,omitempty" toml:"middlewares,omitempty" yaml:"middlewares,omitempty" export:"true"`
	Service     string   `json:"service,omitempty" toml:"service,omitempty" yaml:"service,omitempty" export:"true"`
	Rule        string   `json:"rule,omitempty" toml:"rule,omitempty" yaml:"rule,omitempty"`
	RuleSyntax  string   `json:"ruleSyntax,omitempty" toml:"ruleSyntax,omitempty" yaml:"ruleSyntax,omitempty" export:"true"`
	Priority    int      `json:"priority,omitempty" toml:"priority,omitempty,omitzero" yaml:"priority,omitempty" export:"true"`
	DefaultRule bool     `json:"-" toml:"-" yaml:"-" label:"-" file:"-"`
	Err         []string `json:"error,omitempty"`
	Status      string   `json:"status,omitempty"`
	Using       []string `json:"using,omitempty"`
	Name        string   `json:"name,omitempty"`
	Provider    string   `json:"provider,omitempty"`
}

func TraefikDomains() (map[string][]*Domain, error) {
	traefikUrl, username, password := getTraefikUrl()
	req, err := http.NewRequest("GET", traefikUrl, nil)
	if err != nil {
		return nil, err
	}
	// Set the auth for the request.
	req.SetBasicAuth(username, password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	var routers []RouterInfo
	err = json.Unmarshal(all, &routers)
	if err != nil {
		log.Printf("json解析异常 %s\n", all)
		return nil, fmt.Errorf("json解析异常 %s", all)
	}

	domains := make(map[string]int)
	expr := regexp.MustCompile("Host\\(`([a-zA-Z0-9.\\-]+)`\\)")
	for _, router := range routers {
		if router.Status == "enabled" {
			routerDomain := expr.FindAllStringSubmatch(router.Rule, -1)
			for _, domainArray := range routerDomain {
				domain := domainArray[1]
				domains[domain]++
			}

		}
	}

	domainMap := make(map[string][]*Domain)

	for domain, _ := range domains {
		mainDomain, err := publicsuffix.EffectiveTLDPlusOne(domain)
		if err != nil {
			log.Println("域名解析异常: %s", domain)
			log.Println("异常信息: %s", err)
			continue
		}

		domainLen := len(domain) - len(mainDomain) - 1
		subDomain := "@"
		if domainLen > 0 {
			subDomain = domain[:domainLen]
		}

		domain := &Domain{
			Domain:       mainDomain,
			SubDomain:    subDomain,
			CustomDomain: domain,
		}
		domainMap[mainDomain] = append(domainMap[mainDomain], domain)
	}

	return domainMap, nil
}

func getTraefikUrl() (string, string, string) {

	traefikUrl, username, password := util.ParseUrl(config.GetTraefikHost())
	var router = "/api/http/routers"
	if !strings.HasPrefix(traefikUrl, "http") {
		traefikUrl = "http://" + traefikUrl
	}
	parse, err := url.Parse(traefikUrl + router)
	if err != nil {
		log.Printf("url解析异常:%s %s", traefikUrl, err)
	}
	return parse.String(), username, password

}
