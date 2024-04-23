// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package control

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

type Pauseable interface {
	Pause() error
	Resume() error
}

type server struct {
	port int
	step Pauseable
}

func NewServer(port int, step Pauseable) *server {
	return &server{
		port: port,
		step: step,
	}
}

func (s *server) handler() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.step.Pause(); err != nil {
			fmt.Printf("Warning: failed to pause: %s\n", err.Error())
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
		if err := s.step.Resume(); err != nil {
			fmt.Printf("Warning: failed to resume: %s\n", err.Error())
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
