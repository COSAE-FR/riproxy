package configuration

import (
	"encoding/json"
	"github.com/COSAE-FR/riproxy/domains"
	"github.com/COSAE-FR/riproxy/utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

const defaultProxyPort = 3128
const defaultBindPort = 80

type LoggingConfig struct {
	File  string `yaml:"file" json:"log"`
	Level string
}

type ReverseConfig struct {
	Destination     string
	DestinationIp   net.IP `yaml:"-" json:"-"`
	DestinationPort uint16 `yaml:"-" json:"-"`
	SourceInterface string `yaml:"source_interface" json:"source_interface"`
	SourceIP        net.IP `yaml:"-" json:"-"`
}

func (p *ReverseConfig) check(log *log.Entry, name string, iface string, port uint16) error {
	p.DestinationIp, p.DestinationPort = utils.GetConnection(p.Destination)
	if p.DestinationPort == 0 {
		p.DestinationPort = port
	}
	if len(p.SourceInterface) > 0 {
		iface = p.SourceInterface
	}
	interfaceIP, err := utils.GetIPForInterface(iface)
	if err != nil {
		log.Errorf("cannot get interface ip for: %s '%s'", name, err)
		return err
	}
	p.SourceIP = interfaceIP.IP
	if len(p.DestinationIp) == 0 {
		p.Destination = name
	}
	return nil
}

type InterfaceConfig struct {
	Name                   string
	Bind                   string
	BindIP                 net.IP `yaml:"-" json:"-"`
	BindPort               uint16 `yaml:"-" json:"-"`
	Proxy                  string
	ProxyIP                net.IP      `yaml:"-" json:"-"`
	ProxyPort              uint16      `yaml:"-" json:"-"`
	EnableProxy            bool        `yaml:"-" json:"-"`
	NetworkStrings         []string    `yaml:"networks" json:"networks"`
	InterfaceNetworkDirect bool        `yaml:"direct" json:"direct"`
	Networks               []net.IPNet `yaml:"-" json:"-"`
	Regexp                 []string
	BlockListString        []string `yaml:"block" json:"block"`
	BlockList              domains.DomainTree
	ReverseProxy           map[string]ReverseConfig `yaml:"reverse_proxy" json:"reverse_proxy"`
}

func (i *InterfaceConfig) check(global GlobalConfig, log *log.Entry) error {
	interfaceIP, err := utils.GetIPForInterface(i.Name)
	if err != nil {
		log.Errorf("cannot get interface ip: %s'%s'", i.Name, err)
		return err
	}
	i.BindIP, i.BindPort = utils.GetConnection(i.Bind)
	if i.BindIP == nil {
		i.BindIP = interfaceIP.IP
	}
	if i.BindPort == 0 {
		i.BindPort = defaultBindPort
	}
	i.ProxyIP, i.ProxyPort = utils.GetConnection(i.Proxy)
	if i.ProxyIP == nil {
		i.ProxyIP = interfaceIP.IP
	}
	if i.ProxyPort == 0 {
		i.ProxyPort = defaultProxyPort
	}
	if strings.HasPrefix(i.Proxy, "self") {
		i.EnableProxy = true
	}
	for _, netString := range i.NetworkStrings {
		_, network, err := net.ParseCIDR(netString)
		if err != nil {
			ip, err := utils.GetIPForInterface(netString)
			if err != nil {
				log.Errorf("cannot parse network: %s'%s'", netString, err)
				continue
			}
			_, network, err = net.ParseCIDR(ip.String())
			if err != nil {
				network = ip
			}
		}
		i.Networks = append(i.Networks, *network)
	}
	if i.InterfaceNetworkDirect {
		_, network, err := net.ParseCIDR(interfaceIP.String())
		if err == nil {
			i.Networks = append(i.Networks, *network)
		}
	}
	proxies := make(map[string]ReverseConfig)
	if len(i.ReverseProxy) > 0 {
		for name, config := range i.ReverseProxy {
			err = config.check(log, name, i.Name, i.BindPort)
			if err == nil {
				proxies[name] = config
			}
		}
	}
	i.ReverseProxy = proxies
	if global.BlockByIDN {
		i.BlockList = domains.NewIDNAFromList(i.BlockListString)
	} else {
		i.BlockList = domains.NewFromList(i.BlockListString)
	}
	i.BlockListString = nil
	return nil
}

type GlobalConfig struct {
	BlockByIDN      bool     `yaml:"block_by_idn" json:"block_by_idn"`
	BlockListString []string `yaml:"block" json:"block"`
	BlockList       domains.DomainTree
}

func (c *GlobalConfig) check() error {
	if c.BlockByIDN {
		c.BlockList = domains.NewIDNAFromList(c.BlockListString)
	} else {
		c.BlockList = domains.NewFromList(c.BlockListString)
	}
	c.BlockListString = nil
	return nil
}

type Configuration struct {
	Logging       LoggingConfig
	Global        GlobalConfig `yaml:"global" json:"global"`
	Interfaces    []InterfaceConfig
	Log           *log.Entry `yaml:"-" json:"-"`
	logFileWriter *os.File
	useJson       bool
	path          string
}

func (c *Configuration) setUpLog() {
	if len(c.Logging.Level) == 0 {
		c.Logging.Level = "error"
	}
	logLevel, err := log.ParseLevel(c.Logging.Level)
	if err != nil {
		logLevel = log.ErrorLevel
	}
	log.SetLevel(logLevel)
	logger := log.WithFields(log.Fields{
		"app":       utils.Name,
		"component": "config_loader",
		"version":   utils.Version,
	})
	if len(c.Logging.File) > 0 {
		f, err := os.OpenFile(c.Logging.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err == nil {
			c.logFileWriter = f
			log.SetOutput(f)
		} else {
			logger.Errorf("Cannot open log file %s. Logging to stderr.", c.Logging.File)
			c.logFileWriter = os.Stderr
		}
	} else {
		c.logFileWriter = os.Stderr
	}
	c.Log = logger
}

func (c *Configuration) Read() error {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if c.useJson {
		return c.readJson()
	} else {
		return c.readYaml()
	}
}

func (c *Configuration) readJson() error {
	jsonFile, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer func() {
		_ = jsonFile.Close()
	}()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(byteValue, c)
}

func (c *Configuration) readYaml() error {
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

func (c *Configuration) check() error {
	var interfaces []InterfaceConfig
	for _, i := range c.Interfaces {
		err := i.check(c.Global, c.Log)
		if err != nil {
			c.Log.Errorf("error in %s configuration", i.Name)
		}
		interfaces = append(interfaces, i)
	}
	c.Interfaces = interfaces
	return nil
}

func New(path string, jsonFormat bool) (*Configuration, error) {
	var config Configuration
	config.path = path
	if jsonFormat {
		config.useJson = true
	}
	err := config.Read()
	if err != nil {
		return &config, err
	}
	config.setUpLog()
	err = config.check()
	return &config, err
}
