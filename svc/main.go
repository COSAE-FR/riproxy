package main

import (
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/server"
	"github.com/COSAE-FR/riproxy/utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/hlandau/easyconfig.v1"
	"gopkg.in/hlandau/service.v2"
	"os"
)

type Config struct {
	File string `usage:"configuration file" default:"ripvutils.yml"`
}

type Daemon struct {
	Configuration *configuration.Configuration
	Servers       []server.Server
}

func (d Daemon) Start() error {
	for _, svr := range d.Servers {
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
	return nil
}

func New(cfg Config) (*Daemon, error) {
	config, err := configuration.New(cfg.File, false)
	config.Log.Debugf("%+v", config)
	if err != nil {
		return nil, err
	}
	daemon := Daemon{Configuration: config}
	for _, iface := range daemon.Configuration.Interfaces {
		logger := daemon.Configuration.Log.WithFields(log.Fields{
			"app":       "ripvutils",
			"version":   utils.Version,
			"component": "server",
			"interface": iface.Name,
			"ip":        iface.BindIP.String(),
			"port":      iface.BindPort,
		})
		srv, err := server.New(iface, logger)
		if err != nil {
			return &daemon, err
		}
		daemon.Servers = append(daemon.Servers, *srv)
	}
	return &daemon, nil
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
		QuoteEmptyFields:       true,
	})
	log.SetOutput(os.Stderr)
	logger := log.WithFields(log.Fields{
		"app":       "ripvutils",
		"component": "main",
	})
	cfg := Config{}

	configurator := &easyconfig.Configurator{
		ProgramName: "ripvutils",
	}

	err := easyconfig.Parse(configurator, &cfg)
	if err != nil {
		logger.Fatalf("%v", err)
	}
	logger.Debugf("Started with %#v", cfg)
	service.Main(&service.Info{
		Name:      "ripvutils",
		AllowRoot: true,
		NewFunc: func() (service.Runnable, error) {
			return New(cfg)
		},
	})
}
