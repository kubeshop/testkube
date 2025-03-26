package services

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

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
) executionworkertypes.Worker {
	namespacesConfig := map[string]kubernetesworker.NamespaceConfig{}
	for n, s := range serviceAccountNames {
		namespacesConfig[n] = kubernetesworker.NamespaceConfig{DefaultServiceAccountName: s}
	}
	return executionworker.NewKubernetes(clientSet, processor, kubernetesworker.Config{
		Cluster: kubernetesworker.ClusterConfig{
			Id:               clusterId,
			DefaultNamespace: cfg.TestkubeNamespace,
			DefaultRegistry:  cfg.TestkubeRegistry,
			Namespaces:       namespacesConfig,
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
		FeatureFlags:       featureFlags,
		RunnerId:           runnerId,
		CommonEnvVariables: commonEnvVariables,
	})
}
