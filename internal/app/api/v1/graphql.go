package v1

import (
	"context"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/internal/graphql"
	"github.com/kubeshop/testkube/pkg/log"
	"net/http"
	"time"
)

// RunGraphQLServer runs GraphQL server on go net/http server
// There is an issue with gqlgen and fasthttp server
func (s *TestkubeAPI) RunGraphQLServer(
	ctx context.Context,
	cfg *config.Config,
) error {
	srv := graphql.GetServer(s.Events.Bus, s.ExecutorsClient)

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	httpSrv := &http.Server{Addr: ":" + cfg.GraphqlPort}

	log.DefaultLogger.Infow("running GraphQL server", "port", cfg.GraphqlPort)

	go func() {
		<-ctx.Done()
		// sleep 2 seconds to cover the edge case if SIGTERM or SIGKILL signal occurs before the server is started,
		// so the application does not get stuck
		time.Sleep(2 * time.Second)
		s.Log.Infof("shutting down Testkube GraphQL API server")
		_ = httpSrv.Shutdown(context.Background())
	}()

	return httpSrv.ListenAndServe()
}
