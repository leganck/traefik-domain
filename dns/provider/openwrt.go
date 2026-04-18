package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns/model"
	luci "github.com/leganck/traefik-domain/internal/luci"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

type OpenWRT struct {
	logger *log.Entry
	client *luci.LuciClient
}

func (o *OpenWRT) Init(dnsConf *config.Config, logger *log.Entry) error {
	openWRTHost := config.GetOpenWRTHost()
	if openWRTHost == "" {
		return fmt.Errorf("openwrt host is empty")
	}

	client, err := luci.NewLuciClient(openWRTHost, dnsConf.ID, dnsConf.Secret)
	if err != nil {
		return fmt.Errorf("create openwrt client failed: %v", err)
	}

	o.client = client
	o.logger = logger
	return nil
}

func (o *OpenWRT) List(domain string) ([]*model.Record, error) {
	ctx := context.Background()
	result, err := o.client.UCI(ctx, "get_all", []string{"dhcp"})
	if err != nil {
		return nil, fmt.Errorf("get dhcp config failed: %v", err)
	}

	var records map[string]luci.DnsRecord
	if err := json.Unmarshal([]byte(result), &records); err != nil {
		return nil, fmt.Errorf("unmarshal records failed: %v", err)
	}

	var resultRecords []*model.Record
	for _, record := range records {
		switch record.Type {
		case "domain":
			subDomain := strings.TrimSuffix(record.Name, "."+domain)
			resultRecords = append(resultRecords, &model.Record{
				Name:         subDomain,
				MainDomain:   domain,
				CustomDomain: record.Name,
				Value:        record.IP,
				Type:         "A",
			})
		case "cname":
			subDomain := strings.TrimSuffix(record.CName, "."+domain)
			resultRecords = append(resultRecords, &model.Record{
				Name:         subDomain,
				MainDomain:   domain,
				CustomDomain: record.CName,
				Value:        record.Target,
				Type:         "CNAME",
			})
		}
	}

	return resultRecords, nil
}

func (o *OpenWRT) AddRecord(value, recordType string, list []*traefik.Domain) error {
	if list == nil {
		o.logger.Debugln("no record to add")
		return nil
	}

	ctx := context.Background()
	for _, d := range list {
		var err error
		switch recordType {
		case "A":
			err = o.addA(ctx, d.CustomDomain, value)
		case "CNAME":
			err = o.addCName(ctx, d.CustomDomain, value)
		default:
			o.logger.Warnf("unsupported record type: %s", recordType)
			continue
		}
		if err != nil {
			return fmt.Errorf("add record %s failed: %v", d.CustomDomain, err)
		}
		o.logger.Infof("add record %s -> %s success", d.CustomDomain, value)
	}

	_, err := o.client.UCI(ctx, "commit", []string{"dhcp"})
	return err
}

func (o *OpenWRT) addA(ctx context.Context, name, ip string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if ip == "" {
		return fmt.Errorf("ip is required")
	}

	cfg, err := o.client.UCI(ctx, "add", []string{"dhcp", "domain"})
	if err != nil {
		return err
	}

	if _, err := o.client.UCI(ctx, "set", []string{"dhcp", cfg, "name", name}); err != nil {
		return err
	}

	if _, err := o.client.UCI(ctx, "set", []string{"dhcp", cfg, "ip", ip}); err != nil {
		return err
	}

	if _, err := o.client.UCI(ctx, "set", []string{"dhcp", cfg, "remark", luci.RecordRemark}); err != nil {
		return err
	}

	return nil
}

func (o *OpenWRT) addCName(ctx context.Context, cname, target string) error {
	if cname == "" {
		return fmt.Errorf("cname is required")
	}
	if target == "" {
		return fmt.Errorf("target is required")
	}

	cfg, err := o.client.UCI(ctx, "add", []string{"dhcp", "cname"})
	if err != nil {
		return err
	}

	if _, err := o.client.UCI(ctx, "set", []string{"dhcp", cfg, "cname", cname}); err != nil {
		return err
	}

	if _, err := o.client.UCI(ctx, "set", []string{"dhcp", cfg, "target", target}); err != nil {
		return err
	}

	if _, err := o.client.UCI(ctx, "set", []string{"dhcp", cfg, "remark", luci.RecordRemark}); err != nil {
		return err
	}

	return nil
}

func (o *OpenWRT) UpdateRecord(value string, list []*model.Record) error {
	if len(list) == 0 {
		o.logger.Debugln("no record to update")
		return nil
	}

	ctx := context.Background()
	currentRecords, err := o.List(list[0].MainDomain)
	if err != nil {
		return err
	}

	recordMap := make(map[string]*model.Record)
	for _, r := range currentRecords {
		recordMap[r.Name] = r
	}

	for _, d := range list {
		record, ok := recordMap[d.Name]
		if !ok {
			o.logger.Warnf("record %s not found", d.Name)
			continue
		}

		if record.Type == "A" {
			if err := o.deleteRecordByName(ctx, record.Name); err != nil {
				return err
			}
			if err := o.addA(ctx, record.CustomDomain, d.Value); err != nil {
				return err
			}
		} else if record.Type == "CNAME" {
			if err := o.deleteRecordByName(ctx, record.Name); err != nil {
				return err
			}
			if err := o.addCName(ctx, record.CustomDomain, d.Value); err != nil {
				return err
			}
		}
		o.logger.Infof("update record %s -> %s success", d.Name, value)
	}

	_, err = o.client.UCI(ctx, "commit", []string{"dhcp"})
	return err
}

func (o *OpenWRT) deleteRecordByName(ctx context.Context, name string) error {
	result, err := o.client.UCI(ctx, "get_all", []string{"dhcp"})
	if err != nil {
		return err
	}

	var records map[string]luci.DnsRecord
	if err := json.Unmarshal([]byte(result), &records); err != nil {
		return err
	}

	for cfg, record := range records {
		if record.Remark != luci.RecordRemark {
			continue
		}
		if record.Name == name || record.CName == name {
			_, err := o.client.UCI(ctx, "delete", []string{"dhcp", cfg})
			return err
		}
	}

	return nil
}
