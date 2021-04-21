package utils

import (
	"errors"
	"net"
	"strconv"
)

func GetIPForInterface(interfaceName string) (ipAddress *net.IPNet, err error) {
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if inter.Name == interfaceName {
			if addresses, err := inter.Addrs(); err == nil {
				for _, addr := range addresses {
					switch ip := addr.(type) {
					case *net.IPNet:
						if ip.IP.To4() != nil {
							return ip, nil
						}
					}
				}
			}
		}
	}
	return ipAddress, errors.New("no IP found")
}

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
