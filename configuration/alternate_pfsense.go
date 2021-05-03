// +build pfsense

package configuration

import (
	"encoding/xml"
	"errors"
	"github.com/COSAE-FR/riputils/common/logging"
	"github.com/COSAE-FR/riputils/pfsense/configuration"
	"github.com/COSAE-FR/riputils/pfsense/configuration/sections/packages"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"path/filepath"
)

func NewAlternateConfiguration(path string) (*MainConfiguration, error) {
	if filepath.Ext(path) == ".xml" {
		pfConfig, err := GetConfigurationFromPfSense(path)
		if err == nil {
			pfConfig.Log.Debug("Starting in pfSense mode")
			return pfConfig, nil
		}
	}
	return nil, errors.New("configuration not compatible")
}

const defaultPfSenseLogFile = "/var/log/riproxy.log"

type proxyPackageConfiguration struct {
	packages.BasePackageConfig
	Riproxy      *packages.RiproxyConfig       `xml:"riproxy>config"`
	RiproxyHttp  []packages.RiproxyHttpConfig  `xml:"riproxyhttp>config"`
	RiproxyProxy []packages.RiproxyProxyConfig `xml:"riproxyproxy>config"`
}

type ripSenseConfiguration struct {
	configuration.BaseConfiguration
	Packages proxyPackageConfiguration `xml:"installedpackages"`
}

func resolvePfSenseInterfaces(configuration configuration.InterfaceGetter, interfaces []string, logger *log.Entry) []string {
	result := make([]string, 0)
	for _, interfaceName := range interfaces {
		phy, err := configuration.GetPhysicalInterfaceName(interfaceName)
		if err != nil {
			logger.Errorf("cannot get physical interface for %s in interface list", interfaceName)
			continue
		}
		result = append(result, phy)
	}
	return result
}

func GetConfigurationFromPfSense(path string) (*MainConfiguration, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pfConf := &ripSenseConfiguration{}

	err = xml.Unmarshal(data, &pfConf)
	if err != nil {
		return nil, err
	}

	if pfConf.Packages.Riproxy == nil {
		return nil, errors.New("no riproxy configuration")
	}

	pfProxy := pfConf.Packages.Riproxy

	conf := &MainConfiguration{
		Logging: logging.Config{
			Level: pfProxy.LogLevel,
			File:  defaultPfSenseLogFile,
		},
	}

	conf.setUpLog()

	logger := conf.Log.WithFields(log.Fields{
		"component": "pfsense_loader",
	})

	conf.Defaults = DefaultConfig{
		Http: HttpConfig{
			Port: defaultBindPort,
			Wpad: WpadConfig{
				Proxy:                  "",
				NetworkStrings:         resolvePfSenseInterfaces(pfConf, pfProxy.DirectInterfaces, logger),
				InterfaceNetworkDirect: bool(pfProxy.InterfaceDirect),
			},
		},
		Proxy: ProxyConfig{
			Port:               pfProxy.ProxyPort,
			BlockByIDN:         bool(pfProxy.BlockByIdn),
			AllowHighPorts:     bool(pfProxy.AllowHighPorts),
			AllowLowPorts:      bool(pfProxy.AllowLowPorts),
			BlockIPs:           bool(pfProxy.BlockIps),
			BlockLocalServices: bool(pfProxy.BlockLocalServices),
			BlockListString:    pfProxy.Block,
		},
	}

	conf.Interfaces = map[string]InterfaceConfig{}
	for _, httpConfig := range pfConf.Packages.RiproxyHttp {

		// Get physical interface
		iface, err := pfConf.GetPhysicalInterfaceName(httpConfig.Interface)
		if err != nil {
			logger.Errorf("cannot get physical interface for %s in HTTP config", httpConfig.Interface)
			continue
		}

		// Prepare direct interfaces
		var directs []string
		for _, direct := range httpConfig.DirectInterfaces {
			directIface, err := pfConf.GetPhysicalInterfaceName(direct)
			if err != nil {
				logger.Errorf("cannot get physical interface for %s in HTTP direct interfaces config", direct)
				continue
			}
			directs = append(directs, directIface)
		}

		// Prepare reverse proxies
		reverseProxies := make(map[string]ReverseProxyConfig)
		for _, reverseProxy := range httpConfig.ReverseProxies {
			srcIface := ""
			if len(reverseProxy.Interface) > 0 {
				srcIface, err = pfConf.GetPhysicalInterfaceName(reverseProxy.Interface)
				if err != nil {
					logger.Errorf("cannot get physical interface for %s in reverse proxy config", reverseProxy.Interface)
					srcIface = ""
				}
			}
			peerIP := net.ParseIP(reverseProxy.PeerIP)
			if peerIP == nil {
				logger.Error("cannot configure reverse reverseProxy without peer IP")
				continue
			}
			reverseProxies[reverseProxy.Host] = ReverseProxyConfig{
				PeerIp:          peerIP,
				PeerPort:        reverseProxy.PeerPort,
				SourceInterface: srcIface,
			}
		}

		proxyAddress := ""
		if httpConfig.ExternalProxy {
			proxyAddress = httpConfig.ExternalProxyAddress
		}

		conf.Interfaces[iface] = InterfaceConfig{
			Name: iface,
			Http: InterfaceHttpConfig{
				Enable: bool(httpConfig.Enable),
				HttpConfig: HttpConfig{
					Port: defaultBindPort,
					Wpad: WpadConfig{
						Enable:                 bool(httpConfig.Enable),
						Proxy:                  proxyAddress,
						NetworkStrings:         directs,
						InterfaceNetworkDirect: bool(httpConfig.InterfaceDirect),
					},
				},
				ReverseProxies: reverseProxies,
			},
		}
	}

	for _, proxyConfig := range pfConf.Packages.RiproxyProxy {
		// Get physical interface
		iface, err := pfConf.GetPhysicalInterfaceName(proxyConfig.Interface)
		if err != nil {
			logger.Errorf("cannot get physical interface for %s in proxy config", proxyConfig.Interface)
		}

		finalProxyConf := InterfaceProxyConfig{
			Enable: bool(proxyConfig.Enable),
			ProxyConfig: ProxyConfig{
				Port:               proxyConfig.ProxyPort,
				BlockByIDN:         bool(proxyConfig.BlockByIdn),
				BlockListString:    proxyConfig.Block,
				AllowHighPorts:     bool(proxyConfig.AllowHighPorts),
				AllowLowPorts:      bool(proxyConfig.AllowLowPorts),
				BlockIPs:           bool(proxyConfig.BlockIps),
				BlockLocalServices: bool(proxyConfig.BlockLocalServices),
			},
		}

		interfaceConfig, ok := conf.Interfaces[iface]
		if !ok {
			interfaceConfig.Name = iface
		}
		interfaceConfig.Proxy = finalProxyConf

		conf.Interfaces[iface] = interfaceConfig
	}

	err = conf.check()

	return conf, err
}
