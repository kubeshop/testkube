package control

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
)

type Pauseable interface {
	Pause(time.Time) error
	Resume(ts time.Time) error
}

type server struct {
	port   int
	target Pauseable
}

type ControlServerOptions struct {
	HandlePause  func(ts time.Time) error
	HandleResume func(ts time.Time) error
}

func (p ControlServerOptions) Pause(ts time.Time) error {
	return p.HandlePause(ts)
}

func (p ControlServerOptions) Resume(ts time.Time) error {
	return p.HandleResume(ts)
}

func NewServer(port int, target Pauseable) *server {
	return &server{
		port:   port,
		target: target,
	}
}

func (s *server) handler() *http.ServeMux {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	mux := http.NewServeMux()
	// TODO: Consider "shell" command too for debugging?
	mux.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.target.Pause(time.Now()); err != nil {
			stdoutUnsafe.Warnf("warn: failed to pause: %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.target.Resume(time.Now()); err != nil {
			stdoutUnsafe.Warnf("warn: failed to resume: %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func (s *server) Listen() (func(), error) {
	addr := fmt.Sprintf(":%d", s.port)
	srv := http.Server{Addr: addr, Handler: s.handler()}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	stop := func() {
		_ = srv.Shutdown(context.Background())
	}
	go func() {
		_ = srv.Serve(listener)
	}()
	return stop, err
}
