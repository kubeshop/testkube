package control

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingTarget struct {
	mu    sync.Mutex
	calls []string
}

func (r *recordingTarget) Pause(time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "pause")
	return nil
}

func (r *recordingTarget) Resume(time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "resume")
	return nil
}

func (r *recordingTarget) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func serve(t *testing.T, target Pauseable) net.Conn {
	t.Helper()
	client, serverConn := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	go NewServer(0, target).handler(serverConn)
	return client
}

func TestServer_PauseAndResume(t *testing.T) {
	target := &recordingTarget{}
	conn := serve(t, target)

	for _, packet := range []byte{PausePacket, ResumePacket} {
		_, err := conn.Write([]byte{packet})
		require.NoError(t, err)
		resp := make([]byte, 1)
		_, err = io.ReadFull(conn, resp)
		require.NoError(t, err)
		assert.Equal(t, byte(SuccessPacket), resp[0])
	}
	assert.Equal(t, []string{"pause", "resume"}, target.Calls())
}

func TestServer_ClosesOnUnknownByte(t *testing.T) {
	target := &recordingTarget{}
	conn := serve(t, target)

	// Start of a TLS ClientHello: the record type is followed by 0x03 version bytes.
	go func() { _, _ = conn.Write([]byte{0x16, 0x03, 0x01, 0x03, 0x03}) }()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, err := conn.Read(make([]byte, 1))
	assert.ErrorIs(t, err, io.EOF)
	assert.Empty(t, target.Calls())
}
