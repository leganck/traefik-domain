package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns"
	"github.com/leganck/traefik-domain/traefik"
	"github.com/leganck/traefik-domain/web"
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

	// Initialize web UI switch config
	var switchConfig *web.SwitchConfig
	var httpServer *http.Server
	if conf.WebEnabled {
		switchConfig = web.NewSwitchConfig(conf.WebConfigPath)
		if err := switchConfig.Load(); err != nil {
			log.Warnf("Failed to load switch config: %v", err)
		}
		providers := []string{"dnspod", "adguard", "cloudflare", "openwrt"}
		handler := web.NewHandler(switchConfig, providers)
		httpServer, err = web.StartServer(conf.WebPort, handler)
		if err != nil {
			log.Errorf("Failed to start web server: %v", err)
		} else {
			log.Infof("Web UI enabled on http://localhost:%d", conf.WebPort)
		}
	}

	provider, err := dns.NewDNSProvider(conf, switchConfig)
	if err != nil {
		log.Errorf("create DNS provider error: %v", err)
		panic(err)
	}

	log.Infof("start provider:%s", conf.Name)
	upRecord(provider, conf, switchConfig)

	ticker := time.NewTicker(time.Duration(conf.PollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			upRecord(provider, conf, switchConfig)

		case <-sigChan:
			log.Println("received shutdown signal, exiting...")
			if httpServer != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				httpServer.Shutdown(shutdownCtx)
				shutdownCancel()
			}
			cancel()
			return

		case <-ctx.Done():
			log.Println("exit")
			return
		}
	}

}

func upRecord(provider *dns.Provider, conf *config.Config, switchConfig *web.SwitchConfig) {
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

	// Merge domains to switch config for web UI
	if switchConfig != nil {
		var domainList []string
		for _, domainObjs := range domains {
			for _, d := range domainObjs {
				domainList = append(domainList, d.CustomDomain)
			}
		}
		if err := switchConfig.MergeDomains(domainList); err != nil {
			log.Warnf("Failed to merge domains to switch config: %v", err)
		}
	}

	for k, v := range domains {
		err := provider.AddOrUpdateCname(k, v)
		if err != nil {
			log.Printf("add or update error: %v", err)
		}
	}
}
