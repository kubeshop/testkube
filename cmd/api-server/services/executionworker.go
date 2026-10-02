package services

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
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
		FeatureFlags:                             featureFlags,
		RunnerId:                                 runnerId,
		CommonEnvVariables:                       commonEnvVariables,
		LogAbortedDetails:                        logAbortedDetails,
		AllowLowSecurityFields:                   cfg.AllowLowSecurityFields,
		WorkflowLogsInsecureSkipTLSVerifyBackend: cfg.WorkflowLogsInsecureSkipTLSVerifyBackend,
		TLSRetry: controller.TLSRetryConfig{
			MaxAttempts:  cfg.WorkflowLogsTLSRetryMaxAttempts,
			InitialDelay: cfg.WorkflowLogsTLSRetryInitialDelay,
			MaxDelay:     cfg.WorkflowLogsTLSRetryMaxDelay,
		},
		// Automatically disable resource metrics collection in standalone mode (no API key and no agent registration token configured),
		// as the gRPC control plane connection from execution pods may not be available.
		DisableResourceMetrics:   cfg.TestkubeProAPIKey == "" && cfg.TestkubeProAgentRegToken == "",
		EmptyDirSizeLimit:        cfg.TestkubeEmptyDirSizeLimit,
		StepCacheVolume:          stepCacheVolumeConfig(cfg),
		StepCacheVolumeLocalPath: cfg.TestkubeStepCacheVolumeMountPath,
		DefaultImagePullPolicy:   cfg.TestkubeDefaultImagePullPolicy,
		DefaultRunnerResources: testworkflowconfig.ContainerResourceConfig{
			Requests: testworkflowconfig.ContainerResources{
				CPU:    cfg.TestkubeDefaultRunnerCPURequest,
				Memory: cfg.TestkubeDefaultRunnerMemRequest,
			},
			Limits: testworkflowconfig.ContainerResources{
				CPU:    cfg.TestkubeDefaultRunnerCPULimit,
				Memory: cfg.TestkubeDefaultRunnerMemLimit,
			},
		},
	})
}

// stepCacheVolumeConfig describes the operator's shared step-cache volume, or nil when
// there is none and dependency caches go to the object store whole.
//
// Nil rather than a disabled flag, so that every reader downstream - the processor
// deciding whether to mount anything, the toolkit deciding whether to look - asks one
// question and cannot answer it differently.
func stepCacheVolumeConfig(cfg *config.Config) *testworkflowconfig.StepCacheVolumeConfig {
	// The claim alone decides this, in every mode. commons.StepCacheVolumeEnabled says
	// why, and warns there about the half of the arrangement an agent attached to a
	// Control Plane cannot confirm.
	if !commons.StepCacheVolumeEnabled(cfg) {
		return nil
	}
	return &testworkflowconfig.StepCacheVolumeConfig{ClaimName: cfg.TestkubeStepCacheVolumeClaim}
}
