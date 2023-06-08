package v1

import (
	"context"
	"net"
	"net/http"

	"github.com/kubeshop/testkube/internal/graphql"
	"github.com/kubeshop/testkube/pkg/log"
)

// RunGraphQLServer runs GraphQL server on go net/http server
func (s *TestkubeAPI) RunGraphQLServer(ctx context.Context, port string) error {
	srv := graphql.GetServer(s.Events.Bus, s.ExecutorsClient)

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	httpSrv := &http.Server{Handler: mux}

	log.DefaultLogger.Infow("running GraphQL server", "port", port)

	l, err := net.Listen("tcp", ":"+port)
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
