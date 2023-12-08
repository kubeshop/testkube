package logs

import (
	"context"
	"net/http"
)

// RunHealthCheckHandler is a handler for health check events
// we need HTTP as GRPC probes starts from Kubernetes 1.25
func (ls *LogsService) RunHealthCheckHandler(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ls.httpServer = &http.Server{
		Addr:    ls.httpAddress,
		Handler: mux,
	}

	ls.log.Infow("starting health check handler", "address", ls.httpAddress)
	return ls.httpServer.ListenAndServe()
}
