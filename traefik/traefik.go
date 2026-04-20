package traefik

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/leganck/traefik-domain/dns/model"
	log "github.com/sirupsen/logrus"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

var hostRegex = regexp.MustCompile("Host\\(`([a-zA-Z0-9.\\-]+)`\\)")

type Domain struct {
	MainDomain   string `json:"domain"`
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

func TraefikDomains(host, username, password string) (map[string][]*Domain, error) {
	if host == "" {
		return nil, fmt.Errorf("traefik host is empty")
	}

	traefikUrl := host
	if !strings.HasPrefix(traefikUrl, "http") {
		traefikUrl = "http://" + traefikUrl
	}
	parse, err := url.Parse(traefikUrl + "/api/http/routers")
	if err != nil {
		log.Errorf("failed to parse URL: %s %s", traefikUrl, err)
		return nil, err
	}

	req, err := http.NewRequest("GET", parse.String(), nil)
	if err != nil {
		return nil, err
	}
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Errorf("HTTP request failed: %v", err)
		return nil, err
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Errorf("HTTP status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("read response body failed: %v", err)
		return nil, err
	}
	var routers []RouterInfo
	err = json.Unmarshal(all, &routers)
	if err != nil {
		log.Errorf("failed to parse JSON: %s", all)
		return nil, fmt.Errorf("failed to parse JSON: %s", all)
	}

	domains := make(map[string]int)
	for _, router := range routers {
		if router.Status == "enabled" {
			routerDomain := hostRegex.FindAllStringSubmatch(router.Rule, -1)
			for _, domainArray := range routerDomain {
				domain := domainArray[1]
				domains[domain]++
			}

		}
	}

	domainMap := make(map[string][]*Domain)

	for domain := range domains {
		log.Debugf("traefik domain: %v", domain)
		subDomain, mainDomain, err := model.SplitDomain(domain)
		if err != nil {
			log.Errorf("parse domain : %s  failed : %v", domain, err)
			continue
		}

		domain := &Domain{
			MainDomain:   mainDomain,
			SubDomain:    subDomain,
			CustomDomain: domain,
		}
		domainMap[mainDomain] = append(domainMap[mainDomain], domain)
	}
	return domainMap, nil
}


