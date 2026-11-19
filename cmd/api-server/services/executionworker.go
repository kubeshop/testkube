package services

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

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
		DefaultImagePullPolicy:   cfg.TestkubeDefaultImagePullPolicy,
		ExecuteInSourceNamespace: cfg.GitOpsExecuteInSourceNamespace,
		AgentNamespace:           cfg.TestkubeNamespace,
		// An absolute name, so that no Service in the execution namespace can shadow the agent Service.
		CrossNamespaceLocalApiUrl: fmt.Sprintf("http://%s.%s.svc.%s.:%d", cfg.APIServerFullname, cfg.TestkubeNamespace, clusterDomain(cfg.TestkubeNamespace), cfg.APIServerPort),
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

const (
	resolvConfPath       = "/etc/resolv.conf"
	defaultClusterDomain = "cluster.local"
)

// clusterDomain reads the cluster domain from the DNS search list of the agent pod.
func clusterDomain(namespace string) string {
	f, err := os.Open(resolvConfPath)
	if err != nil {
		return defaultClusterDomain
	}
	defer f.Close()
	return clusterDomainFromResolvConf(f, namespace)
}

// clusterDomainFromResolvConf finds the search domain <namespace>.svc.<domain> and returns <domain>.
func clusterDomainFromResolvConf(r io.Reader, namespace string) string {
	prefix := namespace + ".svc."
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || fields[0] != "search" {
			continue
		}
		for _, domain := range fields[1:] {
			if rest, ok := strings.CutPrefix(strings.TrimSuffix(domain, "."), prefix); ok && rest != "" {
				return rest
			}
		}
	}
	return defaultClusterDomain
}
