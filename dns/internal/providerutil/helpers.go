package providerutil

import (
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
)

func AddDomains(logger *log.Entry, value string, list []*traefik.Domain, add func(*traefik.Domain) (string, error)) error {
	if list == nil {
		logger.Debugf("no record to add")
		return nil
	}

	for _, d := range list {
		resultValue, err := add(d)
		if err != nil {
			logger.Errorf("add record %s %s error: %v", d.CustomDomain, value, err)
			continue
		}
		logger.Infof("add record %s %s success", d.CustomDomain, resultValue)
	}

	logger.Printf("all record add success")
	return nil
}

func UpdateRecords(logger *log.Entry, value string, records []*model.Record, overwrite bool, update func(*model.Record) error) error {
	if len(records) == 0 {
		logger.Debugln("no record to update")
		return nil
	}

	for _, record := range records {
		if !record.Managed && !overwrite {
			logger.Warnf("skip update non-managed record %s", record.CustomDomain)
			continue
		}

		if record.Value == value {
			logger.Infof("record %s %s no need update", record.CustomDomain, record.Value)
			continue
		}

		if err := update(record); err != nil {
			logger.Errorf("update record %s %s error: %v", record.CustomDomain, value, err)
			continue
		}
	}

	logger.Infof("all record update success")
	return nil
}

func DeleteManagedRecords(logger *log.Entry, records []*model.Record, deleteRecord func(*model.Record) error) error {
	if len(records) == 0 {
		logger.Debugln("no record to delete")
		return nil
	}

	for _, record := range records {
		if !record.Managed {
			logger.Warnf("skip delete non-managed record %s", record.CustomDomain)
			continue
		}

		if err := deleteRecord(record); err != nil {
			logger.Errorf("delete record %s error: %v", record.CustomDomain, err)
			continue
		}

		logger.Infof("delete record %s success", record.CustomDomain)
	}

	return nil
}
