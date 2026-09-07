//go:build !linux

package utils

import (
	"fmt"
	"net"
	"runtime"
)

// setReceiveBuffer reports that the receive buffer cannot be honoured on this
// platform. SO_RCVBUF exists elsewhere, but its accounting differs per kernel and
// only the Linux behaviour is specified and tested here, so a requested size is
// refused rather than applied with semantics nobody verified. A size of 0 is the
// zero value of UDPReceiverConfig.ReceiveBuffer and keeps the kernel default, so it
// succeeds everywhere and callers that never set the field are unaffected.
func setReceiveBuffer(_ *net.UDPConn, size int) error {
	if size < 0 {
		return fmt.Errorf("receive buffer size must not be negative, got %d", size)
	}
	if size == 0 {
		return nil
	}
	return fmt.Errorf("receive buffer size %d requested, but it is only supported on linux, not %s; "+
		"set it to 0 to keep the kernel default", size, runtime.GOOS)
}
