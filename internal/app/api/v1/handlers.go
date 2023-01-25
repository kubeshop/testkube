package v1

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/version"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/oauth"
)

const (
	// cliIngressHeader is cli ingress header
	cliIngressHeader = "X-CLI-Ingress"
)

const (
	// mediaTypeJSON is json media type
	mediaTypeJSON = "application/json"
	// mediaTypeYAML is yaml media type
	mediaTypeYAML = "text/yaml"
)

// AuthHandler is auth middleware
func (s *TestkubeAPI) AuthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get(cliIngressHeader, "") != "" {
			token := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization", ""), oauth.AuthorizationPrefix))
			var scopes []string
			if s.oauthParams.Scopes != "" {
				scopes = strings.Split(s.oauthParams.Scopes, ",")
			}

			provider := oauth.NewProvider(s.oauthParams.ClientID, s.oauthParams.ClientSecret, scopes)
			if err := provider.ValidateAccessToken(s.oauthParams.Provider, token); err != nil {
				s.Log.Errorw("error validating token", "error", err)
				return s.Error(c, http.StatusUnauthorized, err)
			}
		}

		return c.Next()
	}
}

// InfoHandler is a handler to get info
func (s *TestkubeAPI) InfoHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(testkube.ServerInfo{
			Commit:    version.Commit,
			Version:   version.Version,
			Namespace: s.Namespace,
		})
	}
}

// RoutesHandler is a handler to get existing routes
func (s *TestkubeAPI) RoutesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var routes []fiber.Route

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

// DebugHandler is a handler to get debug information
func (s *TestkubeAPI) DebugHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		clientSet, err := k8sclient.ConnectToK8s()
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, errors.Wrap(err, "could not connect to cluster"))
		}

		clusterVersion, err := k8sclient.GetClusterVersion(clientSet)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, errors.Wrap(err, "could not get cluster version"))
		}

		apiLogs, err := k8sclient.GetAPIServerLogs(c.UserContext(), clientSet, s.Namespace)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, errors.Wrap(err, "could not get api server logs"))
		}

		operatorLogs, err := k8sclient.GetOperatorLogs(c.UserContext(), clientSet, s.Namespace)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, errors.Wrap(err, "could not get operator logs"))
		}

		executionLogs, err := s.GetLatestExecutionLogs(c.UserContext())
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, errors.Wrap(err, "could not get execution logs"))
		}

		return c.JSON(testkube.DebugInfo{
			ClusterVersion: clusterVersion,
			ApiLogs:        apiLogs,
			OperatorLogs:   operatorLogs,
			ExecutionLogs:  executionLogs,
		})
	}
}
