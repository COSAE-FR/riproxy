package configuration

import (
	"fmt"
	"github.com/COSAE-FR/riproxy/utils"
	"github.com/COSAE-FR/riputils/common"
	"github.com/COSAE-FR/riputils/common/logging"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
)

type DefaultConfig struct {
	Direct LocalNetworks `yaml:",inline"`
	Proxy  ProxyConfig   `yaml:",inline"`
}

func (c *DefaultConfig) check(logger *log.Entry) error {
	if err := c.Proxy.check(nil, nil, logger); err != nil {
		return err
	}
	if err := c.Direct.check(nil, nil, logger); err != nil {
		return err
	}
	return nil
}

type InterfaceConfig struct {
	Name           string                        `yaml:"-"`
	Ip             net.IP                        `yaml:"-"`
	EnableProxy    bool                          `yaml:"enable_proxy"`
	Proxy          ProxyConfig                   `yaml:",inline"`
	Direct         LocalNetworks                 `yaml:",inline"`
	EnableWpad     bool                          `yaml:"enable_wpad"`
	ReverseProxies map[string]ReverseProxyConfig `yaml:"reverse_proxies"`
}

func (i InterfaceConfig) ShouldStartHttp() bool {
	return i.EnableWpad || len(i.ReverseProxies) > 0
}

func (i *InterfaceConfig) check(name string, defaults *DefaultConfig, logger *log.Entry) error {
	interfaceIP, err := common.GetIPForInterface(name)
	if err != nil {
		logger.Errorf("cannot get interface ip: %s'%s'", name, err)
		return err
	}
	i.Name = name
	infos := &interfaceInfo{
		Name: name,
		Ip:   interfaceIP,
	}
	i.Ip = interfaceIP.IP

	// Check Proxy configuration before, we need it to check Http
	err = i.Proxy.check(infos, defaults, logger)
	if err != nil {
		logger.Errorf("cannot prepare Proxy service: %s'%s'", name, err)
		return err
	}
	if i.EnableProxy {
		infos.InterfaceProxy = fmt.Sprintf("%s:%d", i.Ip.String(), i.Proxy.Port)
	}

	// Check the direct networks
	err = i.Direct.check(infos, defaults, logger)
	if err != nil {
		logger.Errorf("cannot prepare direct networks: %s'%s'", name, err)
		return err
	}

	// Check our reverse proxy hosts
	proxies := make(map[string]ReverseProxyConfig)
	if len(i.ReverseProxies) > 0 {
		for name, config := range i.ReverseProxies {
			err = config.check(infos, defaults, logger)
			if err == nil {
				proxies[name] = config
			}
		}
	}
	i.ReverseProxies = proxies
	return nil
}

type LoggingConfig struct {
	logging.Config `yaml:",inline"`
	LogMacAddress  bool `yaml:"log_mac_address"`
}

type MainConfiguration struct {
	Logging       LoggingConfig              `yaml:"logging"`
	Defaults      DefaultConfig              `yaml:"defaults"`
	Interfaces    map[string]InterfaceConfig `yaml:"interfaces"`
	Log           *log.Entry                 `yaml:"-"`
	logFileWriter *os.File
	path          string
}

func (c *MainConfiguration) setUpLog() {
	c.Logging.App = utils.Name
	c.Logging.Version = utils.Version
	c.Logging.Component = "config_loader"
	c.Logging.FileMaxSize = 80
	c.Logging.FileMaxBackups = 10
	c.Log = logging.SetupLog(c.Logging.Config)
}

func (c *MainConfiguration) Read() error {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	yamlFile, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer func() {
		_ = yamlFile.Close()
	}()
	byteValue, err := ioutil.ReadAll(yamlFile)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(byteValue, c)
}

func (c *MainConfiguration) check() error {
	if err := c.Defaults.check(c.Log); err != nil {
		return err
	}
	for name, i := range c.Interfaces {
		err := i.check(name, &c.Defaults, c.Log)
		if err != nil {
			c.Log.Errorf("error in %s configuration", i.Name)
		}
		c.Interfaces[name] = i
	}
	return nil
}

func New(path string) (*MainConfiguration, error) {
	config, err := NewAlternateConfiguration(path)
	if err == nil {
		return config, err
	}
	config = &MainConfiguration{
		path: path,
	}

	err = config.Read()
	if err != nil {
		return config, err
	}
	config.setUpLog()
	err = config.check()
	return config, err
}
