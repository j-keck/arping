package arping

import (
	"errors"
	"fmt"
	"net"
)

func findIpInNetworkFromIface(dstIp net.IP, iface net.Interface) (net.IP, error) {
	if addrs, err := iface.Addrs(); err != nil {
		return nil, err
	} else {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ipnet.Contains(dstIp) {
					return ipnet.IP, nil
				}
			}
		}
		return nil, fmt.Errorf("iface: '%s' can't reach ip: '%s'", iface.Name, dstIp)
	}
}

func findUsableInterfaceForNetwork(dstIp net.IP) (*net.Interface, error) {
	if ifaces, err := net.Interfaces(); err != nil {
		return nil, err
	} else {
		isDown := func(iface net.Interface) bool {
			return iface.Flags&1 == 0
		}

		hasAddressInNetwork := func(iface net.Interface) bool {
			if _, err := findIpInNetworkFromIface(dstIp, iface); err != nil {
				return false
			}
			return true
		}

		verboseLog.Println("search usable interface")
		logIfaceResult := func(msg string, iface net.Interface) {
			verboseLog.Printf("%10s: %6s %18s  %s", msg, iface.Name, iface.HardwareAddr, iface.Flags)
		}

		for _, iface := range ifaces {
			if isDown(iface) {
				logIfaceResult("DOWN", iface)
				continue
			}

			if !hasAddressInNetwork(iface) {
				logIfaceResult("OTHER NET", iface)
				continue
			}

			logIfaceResult("USABLE", iface)
			return &iface, nil
		}
		return nil, errors.New("no usable interface found")
	}
}
