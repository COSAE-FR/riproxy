//go:build pfsense
// +build pfsense

package configuration

import (
	"encoding/xml"
	"errors"
	pfsense2 "github.com/COSAE-FR/riproxy/configuration/pfsense"
	"github.com/COSAE-FR/riputils/common/logging"
	"github.com/COSAE-FR/riputils/pfsense/configuration"
	"github.com/COSAE-FR/riputils/pfsense/configuration/sections/packages"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"path/filepath"
)

func DeleteEmptyString(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

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

const defaultPfSenseLogFile = "/var/log/riproxy/proxy.log"

type proxyPackageConfiguration struct {
	packages.BasePackageConfig
	Riproxy        *pfsense2.RiproxyConfig              `xml:"riproxy>config"`
	RiproxyService []pfsense2.RiproxyServiceConfig      `xml:"riproxyservice>config"`
	RiproxyReverse []pfsense2.RiproxyReverseProxyConfig `xml:"riproxyreverse>config"`
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
		Logging: LoggingConfig{
			Config: logging.Config{
				Level: pfProxy.LogLevel,
				File:  defaultPfSenseLogFile,
			},
			LogMacAddress: true,
		},
	}

	conf.setUpLog()

	logger := conf.Log.WithFields(log.Fields{
		"component": "pfsense_loader",
	})

	var tlsTransparentPort uint16
	if pfProxy.HttpsTransparent {
		if pfProxy.HttpsTransparentPort > 0 {
			tlsTransparentPort = pfProxy.HttpsTransparentPort
		} else {
			tlsTransparentPort = DefaultTlsPort
		}
	}

	// Default configuration
	conf.Defaults = DefaultConfig{
		Direct: LocalNetworks{
			NetworkStrings:         resolvePfSenseInterfaces(pfConf, pfProxy.DirectInterfaces, logger),
			InterfaceNetworkDirect: bool(pfProxy.InterfaceDirect),
		},
		Proxy: ProxyConfig{
			Port:                 pfProxy.ProxyPort,
			BlockByIDN:           bool(pfProxy.BlockByIdn),
			AllowHighPorts:       bool(pfProxy.AllowHighPorts),
			AllowLowPorts:        bool(pfProxy.AllowLowPorts),
			BlockIPs:             bool(pfProxy.BlockIps),
			BlockLocalServices:   bool(pfProxy.BlockLocalServices),
			BlockListString:      DeleteEmptyString(pfProxy.Block),
			HttpTransparent:      bool(pfProxy.HttpTransparent),
			HttpsTransparentPort: tlsTransparentPort,
		},
	}

	conf.Interfaces = map[string]InterfaceConfig{}

	for _, proxyConfig := range pfConf.Packages.RiproxyService {
		if !proxyConfig.EnableProxy {
			continue
		}
		// Get physical interface
		iface, err := pfConf.GetPhysicalInterfaceName(proxyConfig.Interface)
		if err != nil {
			logger.Errorf("cannot get physical interface for %s in proxy config", proxyConfig.Interface)
		}

		var tlsTransparentPort uint16
		if proxyConfig.HttpsTransparent {
			if proxyConfig.HttpsTransparentPort > 0 {
				tlsTransparentPort = proxyConfig.HttpsTransparentPort
			} else {
				tlsTransparentPort = DefaultTlsPort
			}
		}
		finalProxyConf := ProxyConfig{
			Port:                 proxyConfig.ProxyPort,
			BlockByIDN:           bool(proxyConfig.BlockByIdn),
			BlockListString:      DeleteEmptyString(proxyConfig.Block),
			AllowHighPorts:       bool(proxyConfig.AllowHighPorts),
			AllowLowPorts:        bool(proxyConfig.AllowLowPorts),
			BlockIPs:             bool(proxyConfig.BlockIps),
			BlockLocalServices:   bool(proxyConfig.BlockLocalServices),
			HttpTransparent:      bool(proxyConfig.HttpTransparent),
			HttpsTransparentPort: tlsTransparentPort,
		}

		interfaceConfig, ok := conf.Interfaces[iface]
		if !ok {
			interfaceConfig.Name = iface
			interfaceConfig.ReverseProxies = make(map[string]ReverseProxyConfig)
		}
		interfaceConfig.Proxy = finalProxyConf
		interfaceConfig.EnableProxy = bool(proxyConfig.EnableProxy)

		// Honor global WPAD setting
		enableWpad := false
		if pfProxy.EnableWpad {
			enableWpad = true
		} else {
			enableWpad = bool(proxyConfig.EnableWpad)
		}
		interfaceConfig.EnableWpad = enableWpad

		// Prepare direct interfaces
		var directs []string
		for _, direct := range proxyConfig.DirectInterfaces {
			directIface, err := pfConf.GetPhysicalInterfaceName(direct)
			if err != nil {
				logger.Errorf("cannot get physical interface for %s in HTTP direct interfaces config", direct)
				continue
			}
			directs = append(directs, directIface)
		}
		interfaceConfig.Direct.NetworkStrings = directs
		interfaceConfig.Direct.InterfaceNetworkDirect = bool(proxyConfig.InterfaceDirect)

		conf.Interfaces[iface] = interfaceConfig
	}

	for _, reverseConfig := range pfConf.Packages.RiproxyReverse {
		if !reverseConfig.Enable {
			continue
		}

		// Get physical interface
		iface, err := pfConf.GetPhysicalInterfaceName(reverseConfig.Interface)
		if err != nil {
			logger.Errorf("cannot get physical interface for %s in HTTP config", reverseConfig.Interface)
			continue
		}
		peerIP := net.ParseIP(reverseConfig.PeerIP)
		if peerIP == nil {
			logger.Error("cannot configure reverse reverseProxy without peer IP")
			continue
		}

		interfaceConfig, ok := conf.Interfaces[iface]
		if !ok {
			interfaceConfig.Name = iface
			interfaceConfig.ReverseProxies = make(map[string]ReverseProxyConfig)
		}
		srcIface := ""
		if len(reverseConfig.SourceInterface) > 0 {
			srcIface, err = pfConf.GetPhysicalInterfaceName(reverseConfig.SourceInterface)
			if err != nil {
				logger.Errorf("cannot get physical interface for %s in reverse proxy config", reverseConfig.SourceInterface)
				srcIface = ""
			}
		}
		interfaceConfig.ReverseProxies[reverseConfig.Host] = ReverseProxyConfig{
			PeerIp:          peerIP,
			PeerPort:        reverseConfig.PeerPort,
			SourceInterface: srcIface,
		}
		conf.Interfaces[iface] = interfaceConfig
	}

	err = conf.check()

	return conf, err
}
