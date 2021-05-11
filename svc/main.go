package main

import (
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/server"
	"github.com/COSAE-FR/riproxy/utils"
	"github.com/COSAE-FR/riputils/arp"
	"github.com/COSAE-FR/riputils/common/logging"
	log "github.com/sirupsen/logrus"
	"gopkg.in/hlandau/easyconfig.v1"
	"gopkg.in/hlandau/service.v2"
	"time"
)

type Daemon struct {
	Configuration *configuration.MainConfiguration
	LogMacAddress bool
	Servers       []server.Server
}

func (d Daemon) Start() error {
	if d.LogMacAddress {
		d.Configuration.Log.WithField("component", "arp_cache").Debug("Starting ARP cache table auto refresh")
		arp.AutoRefresh(time.Second * 60)
	}
	for _, svr := range d.Servers {
		svr := svr
		err := svr.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d Daemon) Stop() error {
	for _, svr := range d.Servers {
		_ = svr.Stop()
	}
	if d.LogMacAddress {
		d.Configuration.Log.WithField("component", "arp_cache").Debug("Stopping ARP cache table auto refresh")
		arp.StopAutoRefresh()
	}
	return nil
}

func New(cfg Config) (*Daemon, error) {
	config, err := configuration.New(cfg.File)
	if err != nil {
		return nil, err
	}
	daemon := Daemon{Configuration: config}
	daemon.LogMacAddress = config.Logging.LogMacAddress
	for _, iface := range daemon.Configuration.Interfaces {
		logger := daemon.Configuration.Log.WithFields(log.Fields{
			"app":       utils.Name,
			"version":   utils.Version,
			"component": "server",
			"interface": iface.Name,
			"ip":        iface.Ip.String(),
			"port":      configuration.DefaultBindPort,
		})
		srv, err := server.New(iface, &config.Defaults, daemon.LogMacAddress, logger)
		if err != nil {
			return &daemon, err
		}
		daemon.Servers = append(daemon.Servers, *srv)
	}
	return &daemon, nil
}

func main() {
	logger := logging.SetupLog(logging.Config{
		Level:     "error",
		App:       utils.Name,
		Version:   utils.Version,
		Component: "main",
	})

	cfg := Config{}

	configurator := &easyconfig.Configurator{
		ProgramName: utils.Name,
	}

	err := easyconfig.Parse(configurator, &cfg)
	if err != nil {
		logger.Fatalf("%v", err)
	}
	if len(cfg.File) == 0 {
		cfg.File = defaultConfigFileLocation
	}
	logger.Debugf("Starting %s daemon", utils.Name)
	service.Main(&service.Info{
		Name:      utils.Name,
		AllowRoot: true,
		NewFunc: func() (service.Runnable, error) {
			return New(cfg)
		},
	})
}
