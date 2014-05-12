// arping is a native go library to ping a host per arp datagram, or query a host mac address
//
// The currently supported platforms are: Linux and BSD.
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
//       dstIp := net.ParseIP("192.168.1.1")
//       if hwAddr, duration, err := arping.Arping(dstIp); err != nil {
//         fmt.Println(err)
//       } else {
//         fmt.Printf("%s (%s) %d usec\n", dstIp, hwAddr, duration/1000)
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
//       dstIp := net.ParseIP("192.168.1.1")
//       if hwAddr, _, err := arping.Arping(dstIp); err != nil {
//         fmt.Println(err)
//       } else {
//         fmt.Printf("%s is at %s\n", dstIp, hwAddr)
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
//       dstIp := net.ParseIP("192.168.1.1")
//       _, _, err := arping.Arping(dstIp)
//       if err == arping.Timeout {
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
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

var (
	Timeout = errors.New("timeout")

	verboseLog = log.New(ioutil.Discard, "", 0)
	timeout    = time.Duration(500 * time.Millisecond)
)

// sends an arp ping to 'dstIp'
func Arping(dstIp net.IP) (net.HardwareAddr, time.Duration, error) {
	if iface, err := findUsableInterfaceForNetwork(dstIp); err != nil {
		return nil, 0, err
	} else {
		return ArpingOverIface(dstIp, *iface)
	}
}

// sends an arp ping over interface name 'ifaceName' to 'dstIp'
func ArpingOverIfaceByName(dstIp net.IP, ifaceName string) (net.HardwareAddr, time.Duration, error) {
	if iface, err := net.InterfaceByName(ifaceName); err != nil {
		return nil, 0, err
	} else {
		return ArpingOverIface(dstIp, *iface)
	}
}

// sends an arp ping over interface 'iface' to 'dstIp'
func ArpingOverIface(dstIp net.IP, iface net.Interface) (net.HardwareAddr, time.Duration, error) {
	srcMac := iface.HardwareAddr
	srcIp, err := findIpInNetworkFromIface(dstIp, iface)
	if err != nil {
		return nil, 0, err
	}

	broadcastMac := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	request := newArpRequest(srcMac, srcIp, broadcastMac, dstIp)

	if err := initialize(iface); err != nil {
		return nil, 0, err
	}
	defer deinitialize()

	type PingResult struct {
		mac      net.HardwareAddr
		duration time.Duration
		err      error
	}
	pingResultChan := make(chan PingResult)

	go func() {
		// send arp request
		verboseLog.Printf("arping '%s' over interface: '%s' with address: '%s'\n", dstIp, iface.Name, srcIp)
		if sendTime, err := send(request); err != nil {
			pingResultChan <- PingResult{nil, 0, err}
		} else {
			for {
				// receive arp response
				response, receiveTime, err := receive()

				if err != nil {
					pingResultChan <- PingResult{nil, 0, err}
					return
				}

				if response.IsResponseOf(request) {
					duration := receiveTime.Sub(sendTime)
					verboseLog.Printf("process received arp: srcIp: '%s', srcMac: '%s'\n",
						response.SenderIp(), response.SenderMac())
					pingResultChan <- PingResult{response.SenderMac(), duration, err}
					return
				} else {
					verboseLog.Printf("ignore received arp: srcIp: '%s', srcMac: '%s'\n",
						response.SenderIp(), response.SenderMac())
				}
			}
		}
	}()

	select {
	case pingResult := <-pingResultChan:
		return pingResult.mac, pingResult.duration, pingResult.err
	case <-time.After(timeout):
		return nil, 0, Timeout
	}
}

// enable verbose output on stdout
func EnableVerboseLog() {
	verboseLog = log.New(os.Stdout, "", 0)
}

// set ping timeout
func SetTimeout(t time.Duration) {
	timeout = t
}
