package utils

import (
	"net"
	"strconv"
	"testing"

	//"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUDPReceiver(t *testing.T) {
	addr := "::1"
	port, err := getFreeUDPPort()
	require.NoError(t, err)
	t.Logf("starting UDP receiver on %s:%d\n", addr, port)

	r, err := NewUDPReceiver(nil)
	require.NoError(t, err)

	require.NoError(t, r.Start(addr, port, nil))
	sendMessage := func(msg string) error {
		conn, err := net.Dial("udp", net.JoinHostPort(addr, strconv.Itoa(port)))
		if err != nil {
			return err
		}
		_, err = conn.Write([]byte(msg))
		if err != nil {
			if closeErr := conn.Close(); closeErr != nil {
				return closeErr
			}
			return err
		}
		if err := conn.Close(); err != nil {
			return err
		}
		return nil
	}
	require.NoError(t, sendMessage("message"))
	t.Log("sending message\n")
	require.NoError(t, r.Stop())
}

func TestUDPClose(t *testing.T) {
	addr := "::1"
	port, err := getFreeUDPPort()
	require.NoError(t, err)
	t.Logf("starting UDP receiver on %s:%d\n", addr, port)

	r, err := NewUDPReceiver(nil)
	require.NoError(t, err)
	require.NoError(t, r.Start(addr, port, nil))
	require.NoError(t, r.Stop())
	require.NoError(t, r.Start(addr, port, nil))
	require.Error(t, r.Start(addr, port, nil))
	require.NoError(t, r.Stop())
	require.Error(t, r.Stop())
}

func getFreeUDPPort() (int, error) {
	a, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenUDP("udp", a)
	if err != nil {
		return 0, err
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	if err := l.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

// TestUDPReceiverRestart cycles Start/Stop on one receiver. That is the sequence which
// exposed a race between init reassigning r.q and the per-socket closer goroutine
// reading it: Stop's wg.Wait returned while that goroutine was still entering its
// select. Only meaningful under -race.
func TestUDPReceiverRestart(t *testing.T) {
	const addr = "::1"

	port, err := getFreeUDPPort()
	require.NoError(t, err)

	r, err := NewUDPReceiver(&UDPReceiverConfig{Sockets: 2, Workers: 2, QueueSize: 100})
	require.NoError(t, err)

	for range 25 {
		require.NoError(t, r.Start(addr, port, nil))

		// The datagram is what makes this test detect anything: it puts
		// receiveRoutine through a real read and dispatch, which holds the closer
		// goroutine back far enough that it is still entering its select when Stop
		// returns from wg.Wait. Cycling Start/Stop without traffic lets the closer
		// finish first every time, and the window never opens.
		conn, err := net.Dial("udp", net.JoinHostPort(addr, strconv.Itoa(port)))
		require.NoError(t, err)
		_, err = conn.Write([]byte("restart"))
		require.NoError(t, err)
		require.NoError(t, conn.Close())

		require.NoError(t, r.Stop())
	}
}
