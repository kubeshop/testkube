package services

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/identitytoken"
	"github.com/kubeshop/testkube/pkg/log"
)

// CreateIdentityTokenSigner loads the standalone control plane's signing key,
// or returns nil when identity tokens are off.
func CreateIdentityTokenSigner(cfg *config.Config) *identitytoken.Signer {
	if cfg.IdentityTokenPrivateKeyFile == "" {
		return nil
	}
	signer, err := identitytoken.NewSignerFromFile(cfg.IdentityTokenIssuer, cfg.IdentityTokenPrivateKeyFile)
	commons.ExitOnError("Loading identity token signing key", err)
	log.DefaultLogger.Infow("identity tokens enabled", "issuer", signer.Issuer())
	return signer
}

// RegisterIdentityTokenRoutes serves the issuer documents at the root of the
// API, where OIDC discovery puts them relative to the issuer, rather than
// under /v1. The agent API has no authentication, so they are public.
func RegisterIdentityTokenRoutes(mux *fiber.App, signer *identitytoken.Signer) {
	mux.Get(identitytoken.DiscoveryPath, adaptor.HTTPHandlerFunc(signer.HandleDiscovery))
	mux.Get(identitytoken.JWKSPath, adaptor.HTTPHandlerFunc(signer.HandleJWKS))
}
