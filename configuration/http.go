package configuration

import (
	"github.com/COSAE-FR/riputils/common"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
)

type LocalNetworks struct {
	NetworkStrings         []string    `yaml:"direct_networks"`
	InterfaceNetworkDirect bool        `yaml:"direct"`
	Networks               []net.IPNet `yaml:"-"`
}

func (c *LocalNetworks) check(infos *interfaceInfo, defaults *DefaultConfig, logger *log.Entry) error {
	if defaults != nil && defaults.Direct.Networks != nil { // Copy the default networks
		c.Networks = defaults.Direct.Networks[:]
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
	if defaults != nil && defaults.Direct.InterfaceNetworkDirect { // If the defaults is to append network interface
		c.InterfaceNetworkDirect = true
	}
	if infos != nil && c.InterfaceNetworkDirect {
		c.Networks = appendNetwork(c.Networks, *infos.Ip)
	}
	c.NetworkStrings = nil
	return nil
}

type ReverseProxyConfig struct {
	PeerIp          net.IP   `yaml:"peer_ip"`
	PeerPort        uint16   `yaml:"peer_port,omitempty"`
	SourceInterface string   `yaml:"source_interface,omitempty"`
	SourceIP        net.IP   `yaml:"-"`
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
		c.PeerPort = DefaultBindPort
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
