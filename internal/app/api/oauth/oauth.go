package oauth

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/oauth"
)

const (
	// cliIngressHeader is cli ingress header
	cliIngressHeader = "X-CLI-Ingress"
)

type OauthParams struct {
	ClientID     string
	ClientSecret string
	Provider     oauth.ProviderType
	Scopes       string
}

// CreateOAuthHandler is auth middleware
func CreateOAuthHandler(oauthParams OauthParams) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get(cliIngressHeader, "") != "" {
			token := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization", ""), oauth.AuthorizationPrefix))
			var scopes []string
			if oauthParams.Scopes != "" {
				scopes = strings.Split(oauthParams.Scopes, ",")
			}

			provider := oauth.NewProvider(oauthParams.ClientID, oauthParams.ClientSecret, scopes)
			if err := provider.ValidateAccessToken(oauthParams.Provider, token); err != nil {
				log.DefaultLogger.Errorw("error validating token", "error", err)
				return apiutils.SendError(c, http.StatusUnauthorized, err)
			}
		}

		return c.Next()
	}
}
