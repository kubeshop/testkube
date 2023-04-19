//go:generate go run github.com/99designs/gqlgen generate

package resolvers

import (
	"github.com/kubeshop/testkube/internal/graphql/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	ExecutorsService services.ExecutorsService
}
