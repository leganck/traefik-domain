package main

import (
	"github.com/leganck/docker-traefik-domain/config"
	"github.com/leganck/docker-traefik-domain/dns"
	"github.com/leganck/docker-traefik-domain/traefik"
	"golang.org/x/net/context"
	"log"
	"sync"
	"time"
)

var mutex = sync.Mutex{}

func main() {

	conf, err := config.GetConfig()
	if err != nil {
		log.Printf("get config error: %v", err)
		return
	}
	timer := time.Tick(time.Duration(conf.PollInterval) * time.Second)
	ctx := context.Background()
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
