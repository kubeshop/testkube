package v1

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/pkg/analytics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

const (
	// cliIngressHeader is cli ingress header
	cliIngressHeader = "X-CLI-Ingress"
)

const (
	// contentTypeJSON is json content type
	contentTypeJSON = "application/json"
	// contentTypeYAML is yaml content type
	contentTypeYAML = "text/yaml"
)

// HandleEmitterLogs is a handler to emit logs
func (s TestkubeAPI) HandleEmitterLogs() {
	go func() {
		s.Log.Debug("Listening for workers results")
		for resp := range s.EventsEmitter.Responses {
			if resp.Error != nil {
				s.Log.Errorw("got error when sending webhooks", "response", resp)
				continue
			}
			s.Log.Debugw("got webhook response", "response", resp)
		}
	}()
}

// AuthHandler is auth middleware
func (s TestkubeAPI) AuthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get(cliIngressHeader, "") != "" {
			token := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization", ""), oauth.AuthorizationPrefix))
			scopes := []string{}
			if s.oauthParams.Scopes != "" {
				scopes = strings.Split(s.oauthParams.Scopes, ",")
			}

			provider := oauth.NewProvider(s.oauthParams.ClientID, s.oauthParams.ClientSecret, scopes)
			if err := provider.ValidateAccessToken(s.oauthParams.Provider, token); err != nil {
				s.Log.Errorf("error validating token", "error", err)
				return s.Error(c, http.StatusUnauthorized, err)
			}
		}

		return c.Next()
	}
}

// InfoHandler is a handler to get info
func (s TestkubeAPI) InfoHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(testkube.ServerInfo{
			Commit:  api.Commit,
			Version: api.Version,
		})
	}
}

// RoutesHandler is a handler to get existing routes
func (s TestkubeAPI) RoutesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		routes := []fiber.Route{}

		stack := s.Mux.Stack()
		for _, e := range stack {
			for _, s := range e {
				route := *s
				routes = append(routes, route)
			}
		}

		return c.JSON(routes)
	}
}

// AnalyticsHandler is analytics recording middleware
func (s TestkubeAPI) AnalyticsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		go func(host, path, method string) {
			out, err := analytics.SendAnonymousAPIRequestInfo(host, path, api.Version, method, s.ClusterID)
			l := s.Log.With("measurmentId", analytics.TestkubeMeasurementID, "secret", text.Obfuscate(analytics.TestkubeMeasurementSecret), "path", path)
			if err != nil {
				l.Debugw("sending analytics event error", "error", err)
			} else {
				l.Debugw("anonymous info to tracker sent", "output", out)
			}
		}(c.Hostname(), c.Route().Path, c.Method()) // log route path in form /v1/tests/:name

		return c.Next()
	}
}
