package pfsense

import "github.com/COSAE-FR/riputils/pfsense/configuration/helpers"

type RiproxyBaseWpadConfig struct {
	DirectInterfaces helpers.CommaSeparatedList `xml:"directinterfaces"`
	InterfaceDirect  helpers.OnOffBool          `xml:"listeningdirect"`
}

type RiproxyBaseProxyConfig struct {
	ProxyPort            uint16            `xml:"proxyport"`
	AllowHighPorts       helpers.OnOffBool `xml:"allowhigh"`
	AllowLowPorts        helpers.OnOffBool `xml:"allowlow"`
	BlockIps             helpers.OnOffBool `xml:"blockips"`
	BlockLocalServices   helpers.OnOffBool `xml:"blocklocal"`
	HttpTransparent      helpers.OnOffBool `xml:"httptransparent"`
	HttpsTransparent     helpers.OnOffBool `xml:"httpstransparent"`
	HttpsTransparentPort uint16            `xml:"httpstransparentport"`
	BlockByIdn           helpers.OnOffBool `xml:"blockbyidn"`
	Block                []string          `xml:"row>host"`
}

type RiproxyConfig struct {
	Enable     helpers.OnOffBool `xml:"enable"`
	LogLevel   string            `xml:"loglevel"`
	EnableWpad helpers.OnOffBool `xml:"enablewpad"`
	RiproxyBaseWpadConfig
	RiproxyBaseProxyConfig
}

type RiproxyReverseProxyConfig struct {
	Interface       string            `xml:"interface"`
	Enable          helpers.OnOffBool `xml:"enable"`
	Host            string            `xml:"host"`
	PeerIP          string            `xml:"peerip"`
	PeerPort        uint16            `xml:"peerport"`
	SourceInterface string            `xml:"sourceinterface"`
}

type RiproxyServiceConfig struct {
	Interface   string            `xml:"interface"`
	EnableProxy helpers.OnOffBool `xml:"enableproxy"`
	EnableWpad  helpers.OnOffBool `xml:"enablewpad"`
	RiproxyBaseProxyConfig
	RiproxyBaseWpadConfig
}
