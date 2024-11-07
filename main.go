package main

import (
	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"sync"
	"time"
)

var mutex = sync.Mutex{}

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		log.Errorf("config error: %v", err)
		panic(err)
	}
	timer := time.Tick(time.Duration(conf.PollInterval) * time.Second)
	ctx := context.Background()
	log.Infof("start provider:%s", conf.Name)
	upRecord(conf)
	for {
		select {
		case <-timer:
			upRecord(conf)

		case <-ctx.Done():
			log.Println("exit")
			return

		}
	}

}

func upRecord(conf *config.Config) {
	if mutex.TryLock() {
		defer mutex.Unlock()
		provider, err := dns.NewDNSProvider(conf)
		if err != nil {
			panic(err)
		}
		domains, err := traefik.TraefikDomains()
		if err != nil {
			log.Printf("traefik domains error: %v", err)
			return
		}
		for k, v := range domains {
			err := provider.AddOrUpdateCname(k, v)
			if err != nil {
				log.Printf("add or update error: %v", err)
			}

		}
	}
}
