//
// command line arping utility which use the 'arping' library
//
// this utility need raw socket access, please run it
//   under FreeBSD: as root
//   under Linux: as root or with 'cap_net_raw' permission: sudo setcap cap_net_raw+ep <ARPING_PATH>
//
//
// options:
//   -h: print help and exit
//   -v: verbose output
//   -U: unsolicited/gratuitous ARP mode
//   -i: interface name to use
//   -t: timeout - duration with unit - such as 100ms, 500ms, 1s ...
//
//
// exit code:
//    0: target online
//    1: target offline
//    2: error occurred - see command output
//
package main

import (
	"flag"
	"fmt"
	"github.com/j-keck/arping"
	"net"
	"os"
	"time"
)

var (
	helpFlag       = flag.Bool("h", false, "print help and exit")
	verboseFlag    = flag.Bool("v", false, "verbose output")
	gratuitousFlag = flag.Bool("U", false, "unsolicited/gratuitous ARP mode")
	ifaceNameFlag  = flag.String("i", "", "interface name to use - autodetected if omitted")
	timeoutFlag    = flag.Duration("t", 500*time.Millisecond, "timeout - such as 100ms, 500ms, 1s ...")
)

func main() {
	flag.Parse()

	if *helpFlag {
		printHelpAndExit()
	}
	if *verboseFlag {
		arping.EnableVerboseLog()
	}
	arping.SetTimeout(*timeoutFlag)

	if len(flag.Args()) != 1 {
		fmt.Println("Parameter <IP> missing!")
		printHelpAndExit()
	}
	dstIP := net.ParseIP(flag.Arg(0))

	var hwAddr net.HardwareAddr
	var durationNanos time.Duration
	var err error
	if *gratuitousFlag {
		if len(*ifaceNameFlag) > 0 {
			err = arping.GratuitousArpOverIfaceByName(dstIP, *ifaceNameFlag)
		} else {
			err = arping.GratuitousArp(dstIP)
		}
	} else {
		if len(*ifaceNameFlag) > 0 {
			hwAddr, durationNanos, err = arping.PingOverIfaceByName(dstIP, *ifaceNameFlag)
		} else {
			hwAddr, durationNanos, err = arping.Ping(dstIP)
		}
	}

	// ping timeout
	if err == arping.ErrTimeout {
		fmt.Println(err)
		os.Exit(1)
	}

	// ping failed
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	if *gratuitousFlag {
		os.Exit(0)
	}

	// ping success
	durationMicros := durationNanos / 1000

	var durationString string
	if durationMicros > 1000 {
		durationString = fmt.Sprintf("%d,%03d", durationMicros/1000, durationMicros%1000)
	} else {
		durationString = fmt.Sprintf("%d", durationMicros)
	}

	fmt.Printf("%s (%s) %s usec\n", dstIP, hwAddr, durationString)
	os.Exit(0)
}

func printHelpAndExit() {
	fmt.Printf("Usage: %s <FLAGS> <IP>\n\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Printf("\nExit code:\n  0: target online\n  1: target offline\n  2: error occurred\n")
	os.Exit(2)
}
