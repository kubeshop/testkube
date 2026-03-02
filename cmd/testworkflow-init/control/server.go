package control

import (
	"fmt"
	"net"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
)

const (
	PausePacket   = '\u0003'
	ResumePacket  = '\u0004'
	SuccessPacket = '\u0005'
	FailPacket    = '\u0005'
)

type Pauseable interface {
	Pause(time.Time) error
	Resume(ts time.Time) error
}

type server struct {
	port   int
	target Pauseable
}

func NewServer(port int, target Pauseable) *server {
	return &server{
		port:   port,
		target: target,
	}
}

func (s *server) handler(conn net.Conn) {
	defer conn.Close()

	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	buffer := make([]byte, 1)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}
		switch buffer[0] {
		case PausePacket:
			err := s.target.Pause(time.Now())
			if err == nil {
				conn.Write([]byte{SuccessPacket})
			} else {
				stdoutUnsafe.Warnf("warn: failed to pause: %s\n", err.Error())
				conn.Write([]byte{FailPacket})
			}
		case ResumePacket:
			err := s.target.Resume(time.Now())
			if err == nil {
				conn.Write([]byte{SuccessPacket})
			} else {
				stdoutUnsafe.Warnf("warn: failed to resume: %s\n", err.Error())
				conn.Write([]byte{FailPacket})
			}
		}
	}
}

func (s *server) Listen() (func(), error) {
	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	stop := func() {
		_ = listener.Close()
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go s.handler(conn)
		}
	}()
	return stop, err
}
