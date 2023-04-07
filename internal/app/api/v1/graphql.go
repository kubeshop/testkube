package v1

import (
	"context"
	"net"
	"net/http"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/internal/graphql"
	"github.com/kubeshop/testkube/pkg/log"
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
	httpSrv := &http.Server{Addr: ":" + cfg.GraphqlPort, Handler: mux}

	log.DefaultLogger.Infow("running GraphQL server", "port", cfg.GraphqlPort)

	l, err := net.Listen("tcp", httpSrv.Addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		s.Log.Infof("shutting down Testkube GraphQL API server")
		_ = httpSrv.Shutdown(context.Background())
	}()

	return httpSrv.Serve(l)
}
