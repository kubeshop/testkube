package services

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/kubernetesworker"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
)

func CreateExecutionWorker(
	clientSet kubernetes.Interface,
	cfg *config.Config,
	clusterId string,
	runnerId string,
	serviceAccountNames map[string]string,
	processor testworkflowprocessor.Processor,
	featureFlags map[string]string,
	commonEnvVariables []corev1.EnvVar,
	logAbortedDetails bool,
	defaultNamespace string,
) executionworkertypes.Worker {
	namespacesConfig := map[string]kubernetesworker.NamespaceConfig{}
	for n, s := range serviceAccountNames {
		namespacesConfig[n] = kubernetesworker.NamespaceConfig{DefaultServiceAccountName: s}
	}
	insecureRegistries := commons.TrimAndFilterRegistries(cfg.InsecureRegistries)
	return executionworker.NewKubernetes(clientSet, processor, kubernetesworker.Config{
		Cluster: kubernetesworker.ClusterConfig{
			Id:                 clusterId,
			DefaultNamespace:   defaultNamespace,
			DefaultRegistry:    cfg.TestkubeRegistry,
			InsecureRegistries: insecureRegistries,
			Namespaces:         namespacesConfig,
		},
		ImageInspector: kubernetesworker.ImageInspectorConfig{
			CacheEnabled: cfg.EnableImageDataPersistentCache,
			CacheKey:     cfg.ImageDataPersistentCacheKey,
			CacheTTL:     cfg.TestkubeImageCredentialsCacheTTL,
		},
		Connection: testworkflowconfig.WorkerConnectionConfig{
			Url:         cfg.TestkubeProURL,
			AgentID:     cfg.TestkubeProAgentID,
			ApiKey:      cfg.TestkubeProAPIKey, // TODO: Build hash with the runner's API Key?
			SkipVerify:  cfg.TestkubeProSkipVerify,
			TlsInsecure: cfg.TestkubeProTLSInsecure,

			// TODO: Prepare ControlPlane interface for OSS, so we may unify the communication
			LocalApiUrl: fmt.Sprintf("http://%s:%d", cfg.APIServerFullname, cfg.APIServerPort),
		},
		FeatureFlags:           featureFlags,
		RunnerId:               runnerId,
		CommonEnvVariables:     commonEnvVariables,
		LogAbortedDetails:      logAbortedDetails,
		AllowLowSecurityFields: cfg.AllowLowSecurityFields,
		// Automatically disable resource metrics collection in standalone mode (no API key configured),
		// as the gRPC control plane connection from execution pods may not be available.
		DisableResourceMetrics: cfg.TestkubeProAPIKey == "" && cfg.TestkubeProAgentRegToken == "",
	})
}
