package main

import (
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/server"
	"github.com/COSAE-FR/riproxy/utils"
	"github.com/COSAE-FR/riputils/common/logging"
	log "github.com/sirupsen/logrus"
	"gopkg.in/hlandau/easyconfig.v1"
	"gopkg.in/hlandau/service.v2"
)

type Daemon struct {
	Configuration *configuration.MainConfiguration
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
	config, err := configuration.New(cfg.File)
	if err != nil {
		return nil, err
	}
	daemon := Daemon{Configuration: config}
	for _, iface := range daemon.Configuration.Interfaces {
		logger := daemon.Configuration.Log.WithFields(log.Fields{
			"app":       utils.Name,
			"version":   utils.Version,
			"component": "server",
			"interface": iface.Name,
			"ip":        iface.Ip.String(),
			"port":      iface.Http.Port,
		})
		srv, err := server.New(iface, &config.Defaults, logger)
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
