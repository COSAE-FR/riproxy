package utils

import (
	"net"
	"strconv"
)

func GetConnection(connection string) (net.IP, uint16) {
	if len(connection) == 0 {
		return nil, 0
	}
	returnIP := net.IP{}
	var returnPort uint16
	host, portString, err := net.SplitHostPort(connection)
	if err == nil {
		returnIP = net.ParseIP(host)
		port, _ := strconv.ParseUint(portString, 10, 16)
		returnPort = uint16(port)
	} else {
		returnIP = net.ParseIP(connection)
	}
	return returnIP, returnPort
}
