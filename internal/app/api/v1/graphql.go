package v1

import (
	"context"
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

	http.Handle("/graphql", srv)

	log.DefaultLogger.Infow("running GraphQL server", "port", cfg.GraphqlPort)

	return http.ListenAndServe(":"+cfg.GraphqlPort, nil)
}
