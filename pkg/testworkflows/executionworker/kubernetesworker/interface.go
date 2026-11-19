package kubernetesworker

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type NamespaceConfig struct {
	DefaultServiceAccountName string
}

type ClusterConfig struct {
	Id                 string
	DefaultNamespace   string
	DefaultRegistry    string
	InsecureRegistries []string
	Namespaces         map[string]NamespaceConfig
}

type ImageInspectorConfig struct {
	CacheEnabled bool
	CacheKey     string
	CacheTTL     time.Duration
}

type Config struct {
	Cluster                                  ClusterConfig
	ImageInspector                           ImageInspectorConfig
	Connection                               testworkflowconfig.WorkerConnectionConfig
	FeatureFlags                             map[string]string
	RunnerId                                 string
	CommonEnvVariables                       []corev1.EnvVar
	LogAbortedDetails                        bool
	AllowLowSecurityFields                   bool
	WorkflowLogsInsecureSkipTLSVerifyBackend bool
	TLSRetry                                 controller.TLSRetryConfig
	DisableResourceMetrics                   bool
	EmptyDirSizeLimit                        string
	DefaultImagePullPolicy                   string
	DefaultRunnerResources                   testworkflowconfig.ContainerResourceConfig
	StepCacheVolume                          *testworkflowconfig.StepCacheVolumeConfig
	// StepCacheVolumeLocalPath is where the shared step-cache volume is mounted in
	// this process, so that an execution's inbox can be made before its pod runs.
	//
	// It is deliberately not part of StepCacheVolumeConfig: that one is serialized
	// into a pod annotation, and where the agent happens to mount the volume means
	// nothing inside the pod, which reaches its inbox through a subPath instead.
	StepCacheVolumeLocalPath string
}
