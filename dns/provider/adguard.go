package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

type ProviderConfig struct {
	ProviderID  string
	Name        string
	Type        string
	ID          string
	Secret      string
	Host        string
	RecordValue string
}

type AdGuard struct {
	logger     *log.Entry
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

type filteringStatus struct {
	UserRules []string `json:"user_rules"`
}

type setRules struct {
	Rules []string `json:"rules"`
}

func (a *AdGuard) Init(cfg *ProviderConfig, log *log.Entry) error {
	if cfg.Host == "" {
		return fmt.Errorf("adguard host is required")
	}

	baseURL := cfg.Host
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}
	if !strings.HasSuffix(baseURL, "control/") {
		baseURL = baseURL + "control/"
	}

	a.baseURL = baseURL
	a.username = cfg.ID
	a.password = cfg.Secret
	a.logger = log
	a.httpClient = &http.Client{Timeout: 10 * time.Second}
	return nil
}

func (a *AdGuard) getFilteringRules(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"filtering/status", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(a.username, a.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get filtering status failed: status %d", resp.StatusCode)
	}

	var status filteringStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return status.UserRules, nil
}

func (a *AdGuard) saveFilteringRules(ctx context.Context, rules []string) error {
	body := setRules{Rules: rules}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"filtering/set_rules", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.username, a.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("save filtering rules failed: status %d", resp.StatusCode)
	}
	return nil
}

func (p *AdGuard) List(domain string) ([]*model.Record, error) {
	ctx := context.Background()
	rules, err := p.getFilteringRules(ctx)
	if err != nil {
		p.logger.Errorf("getFilteringRules failed: %v", err)
		return nil, fmt.Errorf("getFilteringRules failed: %v", err)
	}

	result := make([]*model.Record, 0)
	for _, rule := range rules {
		if strings.HasPrefix(rule, "@@||") {
			continue
		}

		managed := strings.Contains(rule, "#"+RecordRemark)
		ruleWithoutMarker := rule
		if managed {
			idx := strings.Index(rule, "#"+RecordRemark)
			ruleWithoutMarker = strings.TrimSpace(rule[:idx])
		}

		if strings.HasPrefix(rule, "#") && !strings.HasPrefix(rule, "#"+RecordRemark) {
			parts := strings.SplitN(ruleWithoutMarker, " ", 4)
			if len(parts) < 3 {
				continue
			}
			txtValue := parts[1]
			ruleDomain := parts[2]
			subDomain, mainDomain, err := model.SplitDomain(ruleDomain)
			if err != nil {
				continue
			}
			if mainDomain == domain {
				result = append(result, &model.Record{
					Name:         subDomain,
					MainDomain:   mainDomain,
					CustomDomain: ruleDomain,
					Value:        txtValue,
					Type:         "TXT",
					Managed:      managed,
				})
			}
			continue
		}

		parts := strings.SplitN(ruleWithoutMarker, " ", 3)
		if len(parts) < 2 {
			continue
		}

		ip := parts[0]
		ruleDomain := parts[1]

		subDomain, mainDomain, err := model.SplitDomain(ruleDomain)
		if err != nil {
			continue
		}

		if mainDomain == domain {
			result = append(result, &model.Record{
				Name:         subDomain,
				MainDomain:   mainDomain,
				CustomDomain: ruleDomain,
				Value:        ip,
				Type:         "A",
				Managed:      managed,
			})
		}
	}
	return result, nil
}

func (p *AdGuard) AddRecord(value, recordType string, list []*traefik.Domain) error {
	if list == nil {
		p.logger.Debugf("no record to add")
		return nil
	}

	if recordType != "A" {
		p.logger.Warnf("AdGuard Filtering Rules only supports A records, skipping type %s", recordType)
		return nil
	}

	ctx := context.Background()
	rules, err := p.getFilteringRules(ctx)
	if err != nil {
		return err
	}

	for _, d := range list {
		rule := fmt.Sprintf("%s %s #%s", value, d.CustomDomain, RecordRemark)
		rules = append(rules, rule)
		p.logger.Infof("add rule %s success", rule)
	}

	return p.saveFilteringRules(ctx, rules)
}

func (p *AdGuard) DeleteRecord(list []*model.Record) error {
	if len(list) == 0 {
		p.logger.Debugln("no record to delete")
		return nil
	}

	ctx := context.Background()
	rules, err := p.getFilteringRules(ctx)
	if err != nil {
		return err
	}

	newRules := make([]string, 0)
	for _, rule := range rules {
		keep := true
		for _, d := range list {
			if d.Managed && strings.Contains(rule, d.CustomDomain) && strings.Contains(rule, "#"+RecordRemark) {
				keep = false
				p.logger.Infof("delete rule %s success", rule)
				break
			}
		}
		if keep {
			newRules = append(newRules, rule)
		}
	}

	return p.saveFilteringRules(ctx, newRules)
}

func (p *AdGuard) UpdateRecord(value string, list []*model.Record) error {
	if len(list) == 0 {
		p.logger.Debugln("no record to update")
		return nil
	}

	ctx := context.Background()
	rules, err := p.getFilteringRules(ctx)
	if err != nil {
		return err
	}

	for _, d := range list {
		for i, rule := range rules {
			if strings.Contains(rule, d.CustomDomain) && strings.Contains(rule, "#"+RecordRemark) {
				rules[i] = fmt.Sprintf("%s %s #%s", value, d.CustomDomain, RecordRemark)
				p.logger.Infof("update rule %s -> %s success", d.CustomDomain, value)
				break
			}
		}
	}

	return p.saveFilteringRules(ctx, rules)
}