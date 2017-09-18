package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/caarlos0/env"
	"github.com/docker/go-plugins-helpers/ipam"

	"github.com/ntt-nflex/ipam-driver/db"
	"github.com/ntt-nflex/ipam-driver/handler"
)

type config struct {
	EtcdHosts []string `env:"ETCD_HOSTS" envSeparator:"," envDefault:"http://127.0.0.1:4001"`
	Prefix    string   `env:"ETCD_PREFIX" envDefault:"/nflex/ipam"`
}

func main() {
	cfg := config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Errorf("%+v", err)
	}
	cli := db.NewClient(cfg.EtcdHosts)
	d := handler.NewHandler(cli, cfg.Prefix)
	h := ipam.NewHandler(d)

	log.Info("Starting Up...")
	err = h.ServeUnix("nflex-ipam", 0)
	if err != nil {
		log.Error(err)
	}
}
