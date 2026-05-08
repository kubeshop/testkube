package grpcutils

import (
	"context"

	"golang.org/x/oauth2"
)

// InsecureDangerousTokenSource allows per-RPC credentials to be used without TLS.
// Only use when it is impossible to configure TLS on the transport.
type InsecureDangerousTokenSource struct {
	oauth2.TokenSource
}

func (ts InsecureDangerousTokenSource) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"authorization": token.Type() + " " + token.AccessToken,
	}, nil
}

func (InsecureDangerousTokenSource) RequireTransportSecurity() bool {
	return false
}
