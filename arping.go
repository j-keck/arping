// Package arping is a native go library to ping a host per arp datagram, or query a host mac address
//
// The currently supported platforms are: Linux and BSD.
//
//
// The library requires raw socket access. So it must run as root, or with appropriate capabilities under linux:
// `sudo setcap cap_net_raw+ep <BIN>`.
//
//
// Examples:
//
//   ping a host:
//   ------------
//     package main
//     import ("fmt"; "github.com/j-keck/arping"; "net")
//
//     func main(){
//       dstIP := net.ParseIP("192.168.1.1")
//       if hwAddr, duration, err := arping.Ping(dstIP); err != nil {
//         fmt.Println(err)
//       } else {
//         fmt.Printf("%s (%s) %d usec\n", dstIP, hwAddr, duration/1000)
//       }
//     }
//
//
//   resolve mac address:
//   --------------------
//     package main
//     import ("fmt"; "github.com/j-keck/arping"; "net")
//
//     func main(){
//       dstIP := net.ParseIP("192.168.1.1")
//       if hwAddr, _, err := arping.Ping(dstIP); err != nil {
//         fmt.Println(err)
//       } else {
//         fmt.Printf("%s is at %s\n", dstIP, hwAddr)
//       }
//     }
//
//
//   check if host is online:
//   ------------------------
//     package main
//     import ("fmt"; "github.com/j-keck/arping"; "net")
//
//     func main(){
//       dstIP := net.ParseIP("192.168.1.1")
//       _, _, err := arping.Ping(dstIP)
//       if err == arping.ErrTimeout {
//         fmt.Println("offline")
//       }else if err != nil {
//         fmt.Println(err.Error())
//       }else{
//         fmt.Println("online")
//       }
//     }
//
package arping

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

var (
	// ErrTimeout error
	ErrTimeout = errors.New("timeout")

	// ErrNoUsableInterface no usable interface found
	ErrNoUsableInterface = errors.New("no usable interface found")

	verboseLog = log.New(ioutil.Discard, "", 0)
	timeout    = time.Duration(500 * time.Millisecond)
)

// Ping sends an arp ping to 'dstIP'
func Ping(dstIP net.IP) (net.HardwareAddr, time.Duration, error) {
	if err := validateIP(dstIP); err != nil {
		return nil, 0, err
	}

	iface, err := findUsableInterfaceForNetwork(dstIP)
	if err != nil {
		return nil, 0, err
	}
	return PingOverIface(dstIP, *iface)
}

// PingOverIfaceByName sends an arp ping over interface name 'ifaceName' to 'dstIP'
func PingOverIfaceByName(dstIP net.IP, ifaceName string) (net.HardwareAddr, time.Duration, error) {
	return PingWithOptions(dstIP, WithIfaceByName(ifaceName))
}

// PingOverIface sends an arp ping over interface 'iface' to 'dstIP'
func PingOverIface(dstIP net.IP, iface net.Interface) (net.HardwareAddr, time.Duration, error) {
	return PingWithOptions(dstIP, WithIface(iface))
}

// PingWithOptions sends an arp ping to 'dstIP'
func PingWithOptions(dstIP net.IP, opts ...Option) (net.HardwareAddr, time.Duration, error) {
	if err := validateIP(dstIP); err != nil {
		return nil, 0, err
	}

	var ops = newOptions()
	for _, opt := range opts {
		if err := opt.apply(ops); err != nil {
			return nil, 0, err
		}
	}

	if ops.iface == nil {
		return nil, 0, ErrNoUsableInterface
	}
	iface := *ops.iface
	srcIP := ops.sourceIP
	srcMac := iface.HardwareAddr

	if len(srcIP) == 0 {
		ip, err := findIPInNetworkFromIface(dstIP, iface)
		if err != nil {
			return nil, 0, err
		}
		srcIP = ip
	}

	broadcastMac := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	request := newArpRequest(srcMac, srcIP, broadcastMac, dstIP)

	sock, err := initialize(iface)
	if err != nil {
		return nil, 0, err
	}
	defer sock.deinitialize()

	type PingResult struct {
		mac      net.HardwareAddr
		duration time.Duration
		err      error
	}
	pingResultChan := make(chan PingResult, 1)

	go func() {
		// send arp request
		verboseLog.Printf("arping '%s' over interface: '%s' with address: '%s'\n", dstIP, iface.Name, srcIP)
		if sendTime, err := sock.send(request); err != nil {
			pingResultChan <- PingResult{nil, 0, err}
		} else {
			for {
				// receive arp response
				response, receiveTime, err := sock.receive()

				if err != nil {
					pingResultChan <- PingResult{nil, 0, err}
					return
				}

				if response.IsResponseOf(request) {
					duration := receiveTime.Sub(sendTime)
					verboseLog.Printf("process received arp: srcIP: '%s', srcMac: '%s'\n",
						response.SenderIP(), response.SenderMac())
					pingResultChan <- PingResult{response.SenderMac(), duration, err}
					return
				}

				verboseLog.Printf("ignore received arp: srcIP: '%s', srcMac: '%s'\n",
					response.SenderIP(), response.SenderMac())
			}
		}
	}()

	select {
	case pingResult := <-pingResultChan:
		return pingResult.mac, pingResult.duration, pingResult.err
	case <-time.After(ops.timeout):
		sock.deinitialize()
		return nil, 0, ErrTimeout
	}
}

// GratuitousArp sends an gratuitous arp from 'srcIP'
func GratuitousArp(srcIP net.IP) error {
	return GratuitousArpWithOptions(WithSourceIP(srcIP))
}

// GratuitousArpOverIfaceByName sends an gratuitous arp over interface name 'ifaceName' from 'srcIP'
func GratuitousArpOverIfaceByName(srcIP net.IP, ifaceName string) error {
	return GratuitousArpWithOptions(WithSourceIP(srcIP), WithIfaceByName(ifaceName))
}

// GratuitousArpOverIface sends an gratuitous arp over interface 'iface' from 'srcIP'
func GratuitousArpOverIface(srcIP net.IP, iface net.Interface) error {
	return GratuitousArpWithOptions(WithSourceIP(srcIP), WithIface(iface))
}

// GratuitousArpWithOptions sends an gratuitous arp
func GratuitousArpWithOptions(opts ...Option) error {
	var ops = newOptions()
	for _, opt := range opts {
		if err := opt.apply(ops); err != nil {
			return err
		}
	}

	srcIP := ops.sourceIP
	if err := validateIP(srcIP); err != nil {
		return err
	}

	if ops.iface == nil {
		iface, err := findUsableInterfaceForNetwork(srcIP)
		if err != nil {
			return err
		}
		ops.iface = iface
	}
	iface := *ops.iface

	srcMac := iface.HardwareAddr
	broadcastMac := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	request := newArpRequest(srcMac, srcIP, broadcastMac, srcIP)

	sock, err := initialize(iface)
	if err != nil {
		return err
	}
	defer sock.deinitialize()
	verboseLog.Printf("gratuitous arp over interface: '%s' with address: '%s'\n", iface.Name, srcIP)
	_, err = sock.send(request)
	return err
}

// EnableVerboseLog enables verbose logging on stdout
func EnableVerboseLog() {
	verboseLog = log.New(os.Stdout, "", 0)
}

// SetTimeout sets ping timeout
func SetTimeout(t time.Duration) {
	timeout = t
}

func validateIP(ip net.IP) error {
	// ip must be a valid V4 address
	if len(ip.To4()) != net.IPv4len {
		return fmt.Errorf("not a valid v4 Address: %s", ip)
	}
	return nil
}
