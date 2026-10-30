package credentials

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/expressions"
)

// IdentityTokenRepository mints short-lived identity tokens for the current execution.
type IdentityTokenRepository interface {
	// Get returns a signed identity token for the execution, with the given audience.
	Get(ctx context.Context, audience string) (string, error)
}

type identityTokenRepository struct {
	getClient     func() controlplaneclient.Client
	environmentId string
	executionId   string
}

func NewIdentityTokenRepository(getClient func() controlplaneclient.Client, environmentId string, executionId string) IdentityTokenRepository {
	return &identityTokenRepository{getClient: getClient, environmentId: environmentId, executionId: executionId}
}

func (r *identityTokenRepository) Get(ctx context.Context, audience string) (string, error) {
	return r.getClient().GetIdentityToken(ctx, r.environmentId, r.executionId, audience)
}

// NewIdentityTokenMachine registers the identity_token(audience) function, which returns a short-lived identity
// token for the execution, minted by the Control Plane for the given audience. A workflow can exchange it with a
// cloud provider that trusts the Control Plane as an identity provider, instead of storing long-lived credentials.
// Each call mints a new token, and the observers receive it, so it can be masked in the logs.
func NewIdentityTokenMachine(repository IdentityTokenRepository, observers ...func(audience string, token string)) expressions.Machine {
	return expressions.NewMachine().
		RegisterFunction("identity_token", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			if len(values) != 1 {
				return nil, true, fmt.Errorf(`"identity_token" function expects 1 argument, %d provided`, len(values))
			}
			if !values[0].IsString() {
				return nil, true, fmt.Errorf(`"identity_token" function expects the audience to be a string, %s provided`, values[0].String())
			}
			audience, _ := values[0].StringValue()
			if audience == "" {
				return nil, true, fmt.Errorf(`"identity_token" function expects a non-empty audience`)
			}
			token, err := repository.Get(context.Background(), audience)
			if err != nil {
				return nil, true, fmt.Errorf(`"identity_token" function failed to get a token for audience %q: %w`, audience, err)
			}
			for i := range observers {
				observers[i](audience, token)
			}
			return token, true, nil
		})
}
