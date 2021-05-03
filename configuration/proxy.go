package configuration

import (
	"github.com/COSAE-FR/riproxy/domains"
	"github.com/COSAE-FR/riputils/common"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

type ProxyConfig struct {
	Port               uint16             `yaml:"port,omitempty"`
	BlockByIDN         bool               `yaml:"block_by_idn"`
	BlockListString    []string           `yaml:"block"`
	BlockList          domains.DomainTree `yaml:"-"`
	AllowHighPorts     bool               `yaml:"allow_high_ports"`
	AllowLowPorts      bool               `yaml:"allow_low_ports"`
	BlockIPs           bool               `yaml:"block_ips"`
	BlockLocalServices bool               `yaml:"block_local_services"`
	LocalIps           []net.IP           `yaml:"-"`
	AllowedMethods     []string           `yaml:"allowed_methods"`
}

type InterfaceProxyConfig struct {
	Enable bool
	ProxyConfig
}

func (c *InterfaceProxyConfig) check(infos *interfaceInfo, defaults *DefaultConfig, logger *log.Entry) error {
	if !c.Enable {
		logger.Infof("Not configuring Proxy service on interface %s: disabled by configuration", infos.Name)
		return nil
	}
	return c.ProxyConfig.check(infos, defaults, logger)
}

func (c *ProxyConfig) check(infos *interfaceInfo, defaults *DefaultConfig, logger *log.Entry) error {
	if c.Port == 0 {
		if defaults != nil {
			if defaults.Proxy.Port > 0 {
				c.Port = defaults.Proxy.Port
			} else {
				c.Port = defaultProxyPort
			}
		}
	}
	if defaults != nil {
		if !c.BlockByIDN && defaults.Proxy.BlockByIDN {
			c.BlockByIDN = true
		}
		if !c.AllowHighPorts && defaults.Proxy.AllowHighPorts {
			c.AllowHighPorts = true
		}
		if !c.AllowLowPorts && defaults.Proxy.AllowLowPorts {
			c.AllowLowPorts = true
		}
		if !c.BlockIPs && defaults.Proxy.BlockIPs {
			c.BlockIPs = true
		}
		if !c.BlockLocalServices && defaults.Proxy.BlockLocalServices {
			c.BlockLocalServices = true
		}
	}
	if defaults != nil && c.BlockLocalServices {
		c.LocalIps = common.GetLocalIPs()
	}
	if c.BlockByIDN {
		c.BlockList = domains.NewIDNAFromList(c.BlockListString)
	} else {
		c.BlockList = domains.NewFromList(c.BlockListString)
	}
	c.BlockListString = nil
	if len(c.AllowedMethods) > 0 {
		var allowed []string
		for _, method := range c.AllowedMethods {
			method = strings.ToUpper(method)
			if _, ok := httpMethods[method]; ok {
				allowed = append(allowed, method)
			} else {
				logger.Warnf("Unknown HTTP method %s in proxy configuration, skipping", method)
			}
		}
		c.AllowedMethods = allowed
	} else {
		if defaults != nil {
			c.AllowedMethods = defaults.Proxy.AllowedMethods
		} else {
			for method, isDefaultForProxy := range httpMethods {
				if isDefaultForProxy {
					c.AllowedMethods = append(c.AllowedMethods, method)
				}
			}
		}
	}
	return nil
}
