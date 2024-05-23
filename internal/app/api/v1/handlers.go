package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	// cliIngressHeader is cli ingress header
	cliIngressHeader = "X-CLI-Ingress"
	// mediaTypeJSON is json media type
	mediaTypeJSON = "application/json"
	// mediaTypeYAML is yaml media type
	mediaTypeYAML = "text/yaml"

	// contextCloud is cloud context
	contextCloud = "cloud"
	// contextOSS is oss context
	contextOSS = "oss"
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
	apiContext := contextOSS
	if s.proContext != nil && s.proContext.APIKey != "" {
		apiContext = contextCloud
	}
	var envID, orgID string
	if s.proContext != nil {
		envID = s.proContext.EnvID
		orgID = s.proContext.OrgID
	}

	var executionNamespaces []string
	for namespace := range s.ServiceAccountNames {
		if namespace == s.Namespace {
			continue
		}

		executionNamespaces = append(executionNamespaces, namespace)
	}

	return func(c *fiber.Ctx) error {
		return c.JSON(testkube.ServerInfo{
			Commit:                version.Commit,
			Version:               version.Version,
			Namespace:             s.Namespace,
			Context:               apiContext,
			ClusterId:             s.Config.ClusterID,
			EnvId:                 envID,
			OrgId:                 orgID,
			HelmchartVersion:      s.helmchartVersion,
			DashboardUri:          s.dashboardURI,
			EnableSecretEndpoint:  s.enableSecretsEndpoint,
			DisableSecretCreation: s.disableSecretCreation,
			Features: &testkube.Features{
				LogsV2: s.featureFlags.LogsV2,
			},
			ExecutionNamespaces: executionNamespaces,
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
		errPrefix := "failed to get debug information"
		clientSet, err := k8sclient.ConnectToK8s()
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: could not connect to cluster: %w", errPrefix, err))
		}

		clusterVersion, err := k8sclient.GetClusterVersion(clientSet)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: could not get cluster version: %w", errPrefix, err))
		}

		apiLogs, err := k8sclient.GetAPIServerLogs(c.UserContext(), clientSet, s.Namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: could not get api server logs: %w", errPrefix, err))
		}

		operatorLogs, err := k8sclient.GetOperatorLogs(c.UserContext(), clientSet, s.Namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: could not get operator logs: %w", errPrefix, err))
		}

		executionLogs, err := s.GetLatestExecutionLogs(c.UserContext())
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get execution logs: %w", errPrefix, err))
		}

		return c.JSON(testkube.DebugInfo{
			ClusterVersion: clusterVersion,
			ApiLogs:        apiLogs,
			OperatorLogs:   operatorLogs,
			ExecutionLogs:  executionLogs,
		})
	}
}
