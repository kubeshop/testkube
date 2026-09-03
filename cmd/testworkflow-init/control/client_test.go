package control

import (
	"net"
	"testing"
	"time"
)

func TestClientCloseInterruptsAnEstablishedConnection(t *testing.T) {
	clientConn, peerConn := net.Pipe()
	defer peerConn.Close()
	cancelled := make(chan struct{})
	subject := &client{
		conn: clientConn,
		cancel: func() {
			select {
			case <-cancelled:
			default:
				close(cancelled)
			}
		},
	}

	subject.Close()
	subject.Close() // A timeout path and deferred cleanup may both close it.
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("Close did not cancel the client context")
	}
	if _, err := peerConn.Write([]byte{ResumePacket}); err == nil {
		t.Fatal("writing to the peer unexpectedly succeeded after client Close")
	}
}

func TestClientCloseAcceptsZeroValue(t *testing.T) {
	var zero client
	zero.Close()
	var nilClient *client
	nilClient.Close()
}
