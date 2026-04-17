package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns"
	"github.com/leganck/traefik-domain/traefik"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var mutex = sync.Mutex{}

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		log.Errorf("config error: %v", err)
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	provider, err := dns.NewDNSProvider(conf)
	if err != nil {
		log.Errorf("create DNS provider error: %v", err)
		panic(err)
	}

	log.Infof("start provider:%s", conf.Name)
	upRecord(provider, conf)

	ticker := time.NewTicker(time.Duration(conf.PollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			upRecord(provider, conf)

		case <-sigChan:
			log.Println("received shutdown signal, exiting...")
			cancel()
			return

		case <-ctx.Done():
			log.Println("exit")
			return
		}
	}

}

func upRecord(provider *dns.Provider, conf *config.Config) {
	if !mutex.TryLock() {
		log.Debug("previous sync still running, skipping this round")
		return
	}
	defer mutex.Unlock()

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
