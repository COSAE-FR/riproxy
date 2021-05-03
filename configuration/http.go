package configuration

import (
	"github.com/COSAE-FR/riputils/common"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
)

type WpadConfig struct {
	Enable                 bool
	Proxy                  string      `yaml:"external_proxy"`
	NetworkStrings         []string    `yaml:"direct_networks"`
	InterfaceNetworkDirect bool        `yaml:"direct" json:"direct"`
	Networks               []net.IPNet `yaml:"-" json:"-"`
	Regexp                 []string
}

func (c *WpadConfig) check(infos *interfaceInfo, defaults *DefaultConfig, logger *log.Entry) error {
	if !c.Enable && infos != nil { // if infos is nil, we are configuring the default configuration
		return nil
	}
	if len(c.Proxy) == 0 { // No external proxy defined for this interface or default  WPAD service
		if infos != nil && len(infos.InterfaceProxy) > 0 { // We are configuring an interface WPAD service with an internal proxy
			c.Proxy = infos.InterfaceProxy
		} else if defaults != nil && len(defaults.Http.Wpad.Proxy) > 0 {
			c.Proxy = defaults.Http.Wpad.Proxy
		}
	}
	if c.Proxy == "self" && infos != nil && len(infos.InterfaceProxy) > 0 { // Special value "self"
		c.Proxy = infos.InterfaceProxy
	}
	if infos != nil && len(c.Proxy) == 0 { // If this is an interface configuration and no proxy is defined
		c.Enable = false
	}
	if defaults != nil && defaults.Http.Wpad.Networks != nil { // Copy the default networks
		c.Networks = defaults.Http.Wpad.Networks[:]
	}
	for _, netString := range c.NetworkStrings {
		_, network, err := net.ParseCIDR(netString)
		if err != nil {
			ip, err := common.GetIPForInterface(netString)
			if err != nil {
				logger.Errorf("cannot parse network: %s'%s'", netString, err)
				continue
			}
			_, network, err = net.ParseCIDR(ip.String())
			if err != nil {
				network = ip
			}
		}
		c.Networks = appendNetwork(c.Networks, *network)
	}
	if defaults != nil && defaults.Http.Wpad.InterfaceNetworkDirect { // If the defaults is to append network interface
		c.InterfaceNetworkDirect = true
	}
	if infos != nil && c.InterfaceNetworkDirect {
		c.Networks = appendNetwork(c.Networks, *infos.Ip)
	}
	return nil
}

type HttpConfig struct {
	Port uint16 `yaml:"port"`
	Wpad WpadConfig
}

func (c *HttpConfig) check(infos *interfaceInfo, defaults *DefaultConfig, logger *log.Entry) error {
	if c.Port == 0 {
		if defaults != nil {
			if defaults.Http.Port > 0 {
				c.Port = defaults.Http.Port
			} else {
				c.Port = defaultBindPort
			}
		}
	}
	err := c.Wpad.check(infos, defaults, logger)
	if err != nil {
		return err
	}
	return nil
}

type ReverseProxyConfig struct {
	PeerIp          net.IP   `yaml:"peer_ip"`
	PeerPort        uint16   `yaml:"peer_port,omitempty"`
	SourceInterface string   `yaml:"source_interface,omitempty" json:"source_interface"`
	SourceIP        net.IP   `yaml:"-" json:"-"`
	AllowedMethods  []string `yaml:"allowed_methods"`
}

func (c *ReverseProxyConfig) check(infos *interfaceInfo, defaults *DefaultConfig, logger *log.Entry) error {
	iface := infos.Name
	c.SourceIP = infos.Ip.IP
	if len(c.SourceInterface) > 0 {
		iface = c.SourceInterface
		interfaceIP, err := common.GetIPForInterface(iface)
		if err != nil {
			logger.Errorf("cannot get interface ip for: %s '%s'", iface, err)
			return err
		}
		c.SourceIP = interfaceIP.IP
	}
	if c.PeerPort == 0 {
		c.PeerPort = defaultBindPort
	}
	if len(c.AllowedMethods) > 0 {
		var allowed []string
		for _, method := range c.AllowedMethods {
			method = strings.ToUpper(method)
			if _, ok := httpMethods[method]; ok {
				allowed = append(allowed, method)
			} else {
				logger.Warnf("Unknown HTTP method %s in reverse proxy configuration, skipping", method)
			}
		}
		c.AllowedMethods = allowed
	} else {
		c.AllowedMethods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		}
	}
	return nil
}

type InterfaceHttpConfig struct {
	Enable bool `yaml:"-"`
	HttpConfig
	ReverseProxies map[string]ReverseProxyConfig `yaml:"reverse_proxies"`
}

func (c *InterfaceHttpConfig) check(infos *interfaceInfo, defaults *DefaultConfig, logger *log.Entry) error {
	err := c.HttpConfig.check(infos, defaults, logger)
	if err != nil {
		return err
	}
	proxies := make(map[string]ReverseProxyConfig)
	if len(c.ReverseProxies) > 0 {
		for name, config := range c.ReverseProxies {
			err = config.check(infos, defaults, logger)
			if err == nil {
				proxies[name] = config
			}
		}
	}
	c.ReverseProxies = proxies
	if len(c.ReverseProxies) == 0 && !c.Wpad.Enable { // If not HTTP service is configured, disable HTTP service
		c.Enable = false
	} else {
		c.Enable = true
	}
	return nil
}
