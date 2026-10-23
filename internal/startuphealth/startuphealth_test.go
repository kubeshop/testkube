package startuphealth

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// freePort reserves a port and immediately releases it, so Start can bind the
// same number.
func freePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot reserve a port: %v", err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	if err := lis.Close(); err != nil {
		t.Fatalf("cannot release the reserved port: %v", err)
	}
	return port
}

// get issues one request, retrying while the server comes up, because Start
// returns before its listener is bound.
func get(t *testing.T, port int, path string) *http.Response {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil {
			return resp
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET %s never succeeded: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// bind reports whether the port can be listened on, which is what the real API
// server does right after the startup health server is stopped.
func bind(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	return lis.Close()
}

func TestStartAnswersTheProbes(t *testing.T) {
	port := freePort(t)
	stop := Start(port)
	defer stop()

	resp := get(t, port, "/health")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/health returned %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Anything else has to say why it cannot serve, so a client routed here
	// while the agent is still starting does not read an empty 200 as success.
	other := get(t, port, "/v1/executions")
	defer other.Body.Close()
	if other.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("/v1/executions returned %d, want %d", other.StatusCode, http.StatusServiceUnavailable)
	}
}

// TestStopReleasesThePort covers the handoff that makes this server safe to use
// at all: it holds the API port, so the real API server cannot bind until it has
// let go. A leaked listener here would crash an agent that previously started
// fine.
func TestStopReleasesThePort(t *testing.T) {
	port := freePort(t)
	stop := Start(port)

	// Make sure it really took the port before asking for it back, otherwise
	// this would pass against a server that never started.
	resp := get(t, port, "/health")
	resp.Body.Close()
	if err := bind(port); err == nil {
		t.Fatal("port was bindable while the startup health server was running")
	}

	stop()

	if err := bind(port); err != nil {
		t.Fatalf("API port still held after stop: %v", err)
	}
}

// TestStopBeforeTheListenerBinds covers the fast path where the Control Plane
// answers immediately: stop can run before ListenAndServe has bound, and must
// neither hang nor leave the port taken by a late bind.
func TestStopBeforeTheListenerBinds(t *testing.T) {
	port := freePort(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		Start(port)()
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("stopping the startup health server hung")
	}

	// Give a late ListenAndServe a chance to bind after Shutdown, which is the
	// failure this guards against.
	time.Sleep(200 * time.Millisecond)
	if err := bind(port); err != nil {
		t.Fatalf("API port held after an immediate stop: %v", err)
	}
}

// TestStartOnABusyPortDoesNotBlockStartup records what happens when something
// else already holds the port: Start cannot serve probes and does not report it,
// but it must not panic or stall the caller.
func TestStartOnABusyPortDoesNotBlockStartup(t *testing.T) {
	port := freePort(t)
	squatter, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("cannot occupy the port: %v", err)
	}
	defer squatter.Close()

	done := make(chan func(), 1)
	go func() { done <- Start(port) }()

	select {
	case stop := <-done:
		stop()
	case <-time.After(5 * time.Second):
		t.Fatal("Start blocked when the port was already taken")
	}
}
