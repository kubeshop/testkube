package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/golang/mock/gomock"

	"github.com/kubeshop/testkube/internal/graphql/gen"
	"github.com/kubeshop/testkube/internal/graphql/services"
)

type TestClient struct {
	ctrl             *gomock.Controller
	Client           *client.Client
	ExecutorsService *services.MockExecutorsService
}

func NewTestClient(t *testing.T) *TestClient {
	ctrl := gomock.NewController(t)
	executorsSrv := services.NewMockExecutorsService(ctrl)
	res := &Resolver{
		ExecutorsService: executorsSrv,
	}
	return &TestClient{
		ctrl:             ctrl,
		Client:           client.New(handler.NewDefaultServer(gen.NewExecutableSchema(gen.Config{Resolvers: res}))),
		ExecutorsService: executorsSrv,
	}
}

func (t *TestClient) Finish() {
	t.ctrl.Finish()
}
