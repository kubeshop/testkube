package logs

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
)

// RunHealthCheckHandler is a handler for health check events
// we need HTTP as GRPC probes starts from Kubernetes 1.25
func (ls *LogsService) RunHealthCheckHandler(ctx context.Context) error {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ls.httpServer = &http.Server{
		Addr: ls.httpAddress,
	}

	ls.log.Infow("starting health check handler", "address", ls.httpAddress)
	return ls.httpServer.ListenAndServe()
}

func (ls *LogsService) Shutdown(ctx context.Context) (err error) {
	err = ls.httpServer.Shutdown(ctx)
	if err != nil {
		return err
	}

	// TODO decide how to handle graceful shutdown of consumers

	return nil
}

func (ls *LogsService) WithAddress(address string) *LogsService {
	ls.httpAddress = address
	return ls
}

func (ls *LogsService) WithRandomPort() *LogsService {
	port := rand.Intn(1000) + 17000
	ls.httpAddress = fmt.Sprintf("127.0.0.1:%d", port)
	return ls
}
