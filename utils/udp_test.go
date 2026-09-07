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

func TestUDPReceiverReceiveBuffer(t *testing.T) {
	tests := []struct {
		name string
		cfg  *UDPReceiverConfig
		want int
	}{
		{name: "nil config", cfg: nil, want: 0},
		{name: "unset", cfg: &UDPReceiverConfig{}, want: 0},
		{name: "explicit zero keeps kernel default", cfg: &UDPReceiverConfig{ReceiveBuffer: 0}, want: 0},
		{name: "set", cfg: &UDPReceiverConfig{ReceiveBuffer: 1 << 18}, want: 1 << 18},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewUDPReceiver(tt.cfg)
			require.NoError(t, err)
			require.Equal(t, tt.want, r.receiveBuffer)
		})
	}
}

// TestSetReceiveBufferContract covers the part of setReceiveBuffer that holds on every
// platform, so the validation stays guarded where the Linux-only read-back test does
// not build.
func TestSetReceiveBufferContract(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	require.Error(t, setReceiveBuffer(conn, -1))
	require.NoError(t, setReceiveBuffer(conn, 0))
}

// TestUDPReceiverStartRejectsBadReceiveBuffer guards the wiring, not the validation:
// receive() applies the buffer before close(started) precisely so a failure aborts
// Start instead of being logged and ignored. Downgrading that to a logError call
// leaves every other test in this package passing.
func TestUDPReceiverStartRejectsBadReceiveBuffer(t *testing.T) {
	port, err := getFreeUDPPort()
	require.NoError(t, err)

	r, err := NewUDPReceiver(&UDPReceiverConfig{ReceiveBuffer: -1, QueueSize: 1000})
	require.NoError(t, err)

	require.Error(t, r.Start("::1", port, nil))
}
