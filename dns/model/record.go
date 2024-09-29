package model

import "golang.org/x/net/publicsuffix"

type Record struct {
	Id           string
	Name         string
	Value        string
	Type         string
	MainDomain   string
	CustomDomain string
}

func SplitDomain(customDomain string) (string, string, error) {

	mainDomain, err := publicsuffix.EffectiveTLDPlusOne(customDomain)
	if err != nil {
		return "", "", err
	}
	domainLen := len(customDomain) - len(mainDomain) - 1
	subDomain := ""
	if domainLen > 0 {
		subDomain = customDomain[:domainLen]
	}
	return subDomain, mainDomain, nil
}
