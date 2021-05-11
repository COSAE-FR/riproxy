package server

import (
	"fmt"
	"html/template"
	"net"
)

const WpadTemplate = `
function FindProxyForURL(url, host) {
	// No proxy for internal networks.
	if (isPlainHostName(host){{range .Direct.Networks }} || isInNet(dnsResolve(host), "{{ .IP }}",  "{{ PrintMask .Mask }}"){{end}}) {
		return "DIRECT";
	}

	// send to proxy
	return "PROXY {{.Proxy.Connection}}";
}
`

var WpadPaths = map[string]bool{
	"/proxy.pac": true,
	"/wpad.dat":  true,
	"/wpad.da":   true,
}

var wpadFile template.Template

func init() {
	tmpl, err := template.New("wpad").Funcs(template.FuncMap{"PrintMask": func(mask net.IPMask) string { return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3]) }}).Parse(WpadTemplate)
	if err != nil {
		panic(err)
	}
	wpadFile = *tmpl
}
