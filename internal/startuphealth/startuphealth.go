// Package startuphealth serves the API port while the agent waits for the
// Control Plane to become reachable.
//
// It lives outside cmd/api-server because that package calls flag.Parse() from
// init(), which runs before testing registers its own flags and so makes the
// package impossible to build a test binary for.
package startuphealth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/kubeshop/testkube/pkg/log"
)

// Start answers /health on the API port and returns a function that stops it.
// The returned function must run before the real API server binds the same port.
//
// Without this, retrying instead of exiting only delays the crash: the Helm
// startup probe polls /health on this port and restarts the container once it
// runs out of attempts (200 x 3s in the testkube-api chart, 100 x 1s in
// testkube-runner), so a Control Plane outage longer than that still restarts an
// agent that is deliberately waiting.
//
// Note that all three chart probes share this one endpoint, so answering the
// startup probe also marks the Pod ready before startup has finished. Everything
// other than /health returns 503 so a client that gets routed here sees why.
func Start(port int) func() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "testkube agent is starting up", http.StatusServiceUnavailable)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.DefaultLogger.Warnw("startup health server stopped", "error", err)
		}
	}()

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.DefaultLogger.Warnw("cannot shut down startup health server", "error", err)
		}
	}
}
