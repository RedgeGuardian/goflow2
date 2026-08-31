package utils

import (
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// sockoptRcvbuf reads SO_RCVBUF back off the socket. Linux reports twice the granted
// size, because it doubles the request to cover its own per-packet bookkeeping.
func sockoptRcvbuf(t *testing.T, conn *net.UDPConn) int {
	t.Helper()

	raw, err := conn.SyscallConn()
	require.NoError(t, err)

	var (
		size   int
		optErr error
	)
	require.NoError(t, raw.Control(func(fd uintptr) {
		size, optErr = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	}))
	require.NoError(t, optErr)

	return size
}

// TestSetReceiveBuffer checks the option actually reaches the socket. Asserting only
// that setReceiveBuffer returns nil would still pass with the setsockopt call removed:
// the kernel clamps an oversized request to net.core.rmem_max silently instead of
// failing, so a nil error proves nothing on its own.
func TestSetReceiveBuffer(t *testing.T) {
	// Well under net.core.rmem_max on any realistic host, so the kernel grants the
	// request in full and the doubled read-back is exact rather than clamped.
	const size = 1 << 16

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	before := sockoptRcvbuf(t, conn)

	require.NoError(t, setReceiveBuffer(conn, size))
	require.Equal(t, 2*size, sockoptRcvbuf(t, conn))

	// Zero must skip the call rather than forward it: SetReadBuffer(0) shrinks the
	// socket to SOCK_MIN_RCVBUF instead of leaving the current size alone.
	require.NoError(t, setReceiveBuffer(conn, 0))
	require.Equal(t, 2*size, sockoptRcvbuf(t, conn))

	// A negative size is a caller bug and must not reach the kernel, which reads it
	// unsigned and would quietly apply net.core.rmem_max instead of failing.
	require.Error(t, setReceiveBuffer(conn, -1))
	require.Equal(t, 2*size, sockoptRcvbuf(t, conn))

	require.NotEqual(t, before, 2*size, "test would be vacuous if the default already matched")
}

func TestUDPReceiveBufferStart(t *testing.T) {
	addr := "::1"
	port, err := getFreeUDPPort()
	require.NoError(t, err)
	t.Logf("starting UDP receiver on %s:%d\n", addr, port)

	// Stays under any plausible net.core.rmem_max; the kernel clamps silently anyway.
	r, err := NewUDPReceiver(&UDPReceiverConfig{ReceiveBuffer: 1 << 18, QueueSize: 1000})
	require.NoError(t, err)

	require.NoError(t, r.Start(addr, port, nil))
	require.NoError(t, r.Stop())
}
