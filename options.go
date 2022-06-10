package arping

import (
	"net"
	"time"
)

type options struct {
	iface    *net.Interface
	sourceIP net.IP
	timeout  time.Duration
}

func newOptions() *options {
	return &options{timeout: timeout}
}

// Option configures source ip, timeout, Interface for PingWithOptions
type Option interface {
	apply(*options) error
}

type ifaceName string

func (n ifaceName) apply(opts *options) error {
	iface, err := net.InterfaceByName(string(n))
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	opts.iface = iface
	return nil
}

// WithIfaceByName sends an arp ping over interface name 'name'
func WithIfaceByName(name string) Option {
	return ifaceName(name)
}

type netInterface net.Interface

func (n netInterface) apply(opts *options) error {
	opts.iface = (*net.Interface)(&n)
	return nil
}

// WithIface sends an arp ping over interface 'iface'
func WithIface(iface net.Interface) Option {
	return netInterface(iface)
}

type netIP net.IP

func (n netIP) apply(opts *options) error {
	opts.sourceIP = net.IP(n)
	return nil
}

// WithSourceIP sends an arp with source ip address
func WithSourceIP(srcIP net.IP) Option {
	return netIP(srcIP)
}

type duration time.Duration

func (n duration) apply(opts *options) error {
	opts.timeout = time.Duration(n)
	return nil
}

// WithTimeout sets ping timeout
// not works with GratuitousArpWithOptions
func WithTimeout(timeout time.Duration) Option {
	return duration(timeout)
}
