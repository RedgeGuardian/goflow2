//go:build !linux

package utils

import (
	"net"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSetReceiveBufferUnsupported checks that a platform whose SO_RCVBUF accounting is
// not verified here refuses a requested size instead of silently ignoring it.
func TestSetReceiveBufferUnsupported(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	err = setReceiveBuffer(conn, 1<<16)
	require.Error(t, err)
	require.ErrorContains(t, err, "65536")
	require.ErrorContains(t, err, runtime.GOOS)
}
