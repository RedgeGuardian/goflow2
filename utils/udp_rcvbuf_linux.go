package utils

import (
	"fmt"
	"net"
)

// setReceiveBuffer requests size bytes of socket receive buffer (SO_RCVBUF) on conn.
// A size of 0 keeps the kernel default by skipping the call: SetReadBuffer(0) does
// not mean "leave it alone", it shrinks the buffer to SOCK_MIN_RCVBUF. The kernel
// doubles an accepted value for its own bookkeeping and silently clamps it to
// net.core.rmem_max, so a nil error does not mean the full size was granted.
func setReceiveBuffer(conn *net.UDPConn, size int) error {
	if size < 0 {
		// Rejected rather than ignored: the kernel reads the value unsigned, so a
		// negative size does not fail, it quietly applies net.core.rmem_max instead.
		return fmt.Errorf("receive buffer size must not be negative, got %d", size)
	}
	if size == 0 {
		return nil
	}
	if err := conn.SetReadBuffer(size); err != nil {
		return fmt.Errorf("set receive buffer %d: %w", size, err)
	}
	return nil
}
