package main

const defaultConfigFileLocation = "/etc/riproxy/riproxy.yml"

type Config struct {
	File string `usage:"configuration file" default:"/etc/riproxy/riproxy.yml"`
}
