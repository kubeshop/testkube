package deprecatedv1

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/kubeshop/testkube/internal/graphql"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
)

// RunGraphQLServer runs GraphQL server on go net/http server
func (s *DeprecatedTestkubeAPI) RunGraphQLServer(ctx context.Context, eventBus bus.Bus) error {
	srv := graphql.GetServer(eventBus, s.DeprecatedClients.Executors())

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	httpSrv := &http.Server{Handler: mux}

	log.DefaultLogger.Infow("running GraphQL server", "port", s.graphqlPort)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.graphqlPort))
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
