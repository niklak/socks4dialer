package socks4dialer

import (
	"fmt"
	"net"
	"strconv"
)

func lookupIPV4(host string) (ip net.IP, err error) {

	ips, err := net.LookupIP(host)
	if err != nil {
		return
	}
	for _, it := range ips {
		ipv4 := it.To4()
		if ipv4 != nil {
			ip = ipv4
			return
		}
	}
	if ip == nil {
		err = fmt.Errorf("unable to resolve host: %s", host)
	}
	return
}

func socksSplitHostPort(address string) (string, int, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	portnum, err := strconv.Atoi(port)
	if err != nil {
		return "", 0, err
	}
	if 1 > portnum || portnum > 0xffff {
		return "", 0, fmt.Errorf("port number out of range %s", port)
	}
	return host, portnum, nil
}
