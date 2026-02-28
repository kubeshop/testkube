package control

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/pkg/errors"
)

type client struct {
	cancel context.CancelFunc
	conn   net.Conn
	mu     sync.Mutex
}

func NewClient(ctx context.Context, address string, port int) (*client, error) {
	dialCtx, cancel := context.WithCancel(ctx)
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		cancel()
		return nil, err
	}
	return &client{conn: conn, cancel: cancel}, nil
}

func (c *client) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.conn.Write([]byte{PausePacket})
	if err != nil {
		return errors.Wrap(err, "failed to send pause packet")
	}
	buffer := make([]byte, 1)
	_, err = c.conn.Read(buffer)
	if err != nil {
		return errors.Wrap(err, "failed to read response for pause packet")
	}
	if buffer[0] != SuccessPacket {
		return errors.New("received error from the pause controller")
	}
	return nil
}

func (c *client) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.conn.Write([]byte{ResumePacket})
	if err != nil {
		return errors.Wrap(err, "failed to send resume packet")
	}
	buffer := make([]byte, 1)
	_, err = c.conn.Read(buffer)
	if err != nil {
		return errors.Wrap(err, "failed to read response for resume packet")
	}
	if buffer[0] != SuccessPacket {
		return errors.New("received error from the resume controller")
	}
	return nil
}

func (c *client) Close() {
	c.cancel()
	_ = c.conn.Close()
}
