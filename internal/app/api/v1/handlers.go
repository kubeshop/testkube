package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	// mediaTypeJSON is json media type
	mediaTypeJSON = "application/json"
	// mediaTypeYAML is yaml media type
	mediaTypeYAML    = "text/yaml"
	mediaTypeYAMLAlt = "application/yaml"
	// mediaTypePlainText is plain text media type
	mediaTypePlainText = "text/plain"

	// contextCloud is cloud context
	contextCloud = "cloud"
	// contextOSS is oss context
	contextOSS = "oss"
)

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
			ClusterId:             s.ClusterID,
			EnvId:                 envID,
			OrgId:                 orgID,
			HelmchartVersion:      s.helmchartVersion,
			DashboardUri:          s.proContext.DashboardURI,
			EnableSecretEndpoint:  s.secretConfig.List,
			DisableSecretCreation: !s.secretConfig.AutoCreate,
			Secret:                &s.secretConfig,
			ExecutionNamespaces:   executionNamespaces,
			DockerImageVersion:    s.dockerImageVersion,
		})
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

		return c.JSON(testkube.DebugInfo{
			ClusterVersion: clusterVersion,
			ApiLogs:        apiLogs,
			OperatorLogs:   operatorLogs,
		})
	}
}

// Warn writes RFC-7807 json problem to response
func (s *TestkubeAPI) Warn(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	return apiutils.SendWarn(c, status, err, context...)
}

// Error writes RFC-7807 json problem to response
func (s *TestkubeAPI) Error(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	return apiutils.SendError(c, status, err, context...)
}

func (s *TestkubeAPI) NotImplemented(c *fiber.Ctx) error {
	return s.Error(c, http.StatusNotImplemented, errors.New("not implemented yet"))
}

func (s *TestkubeAPI) BadGateway(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *TestkubeAPI) InternalError(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *TestkubeAPI) BadRequest(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *TestkubeAPI) NotFound(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *TestkubeAPI) ClientError(c *fiber.Ctx, prefix string, err error) error {
	if apiutils.IsNotFound(err) {
		return s.NotFound(c, prefix, "client not found", err)
	}
	return s.BadGateway(c, prefix, "client problem", err)
}
