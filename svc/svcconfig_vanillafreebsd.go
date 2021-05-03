// +build freebsd,!pfsense

package main

const defaultConfigFileLocation = "/usr/local/etc/riproxy/riproxy.yml"

type Config struct {
	File string `usage:"configuration file" default:"/usr/local/etc/riproxy/riproxy.yml"`
}
