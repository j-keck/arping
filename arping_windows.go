// windows currently not supported.
// dummy implementation to prevent compilation errors under windows
package arping

import (
	"errors"
	"net"
	"time"
)

var windowsNotSupported = errors.New("arping under windows not supported")

func initialize(iface net.Interface) error {
	return windowsNotSupported
}

func send(request arpDatagram) (time.Time, error) {
	return new(time.Time), windowsNotSupported
}

func receive() (arpDatagram, time.Time, error) {
	return new(arpDatagram), new(time.Time), windowsNotSupported
}

func deinitialize() error {
	return windowsNotSupported
}
