package env

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	config2 "github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	config3 "github.com/kubeshop/testkube/internal/config"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/cache"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	phttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	proContextMu    sync.Mutex
	proContextCache *cloud.ProContextResponse

	cloudMu     sync.Mutex
	cloudClient cloud.TestKubeCloudAPIClient
	cloudConn   *grpc.ClientConn
)

func loadProContext() (*cloud.ProContextResponse, error) {
	proContextMu.Lock()
	defer proContextMu.Unlock()

	if proContextCache != nil {
		return proContextCache, nil
	}

	cfg := config2.Config()
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(map[string]string{
		"api-key":         cfg.Worker.Connection.ApiKey,
		"organization-id": cfg.Execution.OrganizationId,
		"environment-id":  cfg.Execution.EnvironmentId,
		"execution-id":    cfg.Execution.Id,
		"agent-id":        cfg.Worker.Connection.AgentID,
	}))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cloudInternal, err := CloudInternal()
	if err != nil {
		return nil, err
	}
	proContext, err := cloudInternal.GetProContext(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	proContextCache = proContext
	return proContext, nil
}

func IsExternalStorage() bool {
	return capabilities.Enabled(GetCapabilities(), capabilities.CapabilityCloudStorage)
}

func GetCapabilities() []*cloud.Capability {
	proContext, err := loadProContext()
	if err != nil {
		return nil
	}
	return proContext.Capabilities
}

func HasJunitSupport() bool {
	return config2.JUnitParserEnabled() || capabilities.Enabled(GetCapabilities(), capabilities.CapabilityJUnitReports)
}

func KubernetesConfig() *rest.Config {
	c, err := rest.InClusterConfig()
	if err != nil {
		var fsErr error
		c, fsErr = k8sclient.GetK8sClientConfig()
		if fsErr != nil {
			ui.Fail(fmt.Errorf("couldn't find Kubernetes config: %w and %w", err, fsErr))
		}
	}
	c.QPS = float32(math.Max(float64(c.QPS), 30))
	c.Burst = int(math.Max(float64(c.Burst), 50))
	return c
}

func Kubernetes() *kubernetes.Clientset {
	c, err := kubernetes.NewForConfig(KubernetesConfig())
	if err != nil {
		ui.Fail(fmt.Errorf("couldn't instantiate Kubernetes client: %w", err))
	}
	return c
}

func ImageInspector() imageinspector.Inspector {
	clientSet := Kubernetes()
	secretClient := &secret.Client{ClientSet: clientSet, Namespace: config2.Namespace(), Log: log.DefaultLogger}
	configMapClient := &configmap.Client{ClientSet: clientSet, Namespace: config2.Namespace(), Log: log.DefaultLogger}
	inspectorStorages := []imageinspector.Storage{imageinspector.NewMemoryStorage()}
	if config2.Config().Worker.ImageInspectorPersistenceEnabled {
		configmapStorage := imageinspector.NewConfigMapStorage(configMapClient, config2.Config().Worker.ImageInspectorPersistenceCacheKey, true)
		_ = configmapStorage.CopyTo(context.Background(), inspectorStorages[0].(imageinspector.StorageTransfer))
		inspectorStorages = append(inspectorStorages, configmapStorage)
	}
	return imageinspector.NewInspector(
		config2.Config().Worker.DefaultRegistry,
		imageinspector.NewCraneFetcher(config2.Config().Worker.InsecureRegistries...),
		imageinspector.NewSecretFetcher(secretClient, cache.NewInMemoryCache[*corev1.Secret](), imageinspector.WithSecretCacheTTL(config2.Config().Worker.ImageInspectorPersistenceCacheTTL)),
		inspectorStorages...,
	)
}

func Testkube() client.Client {
	uri, err := url.Parse(config2.Config().Worker.Connection.LocalApiUrl)
	host := config.APIServerName
	port := config.APIServerPort
	if err == nil {
		host = uri.Hostname()
		portStr, _ := strconv.ParseInt(uri.Port(), 10, 32)
		port = int(portStr)
	}
	if config2.UseProxy() {
		return client.NewProxyAPIClient(Kubernetes(), client.NewAPIConfig(config2.Namespace(), host, port))
	}
	httpClient := phttp.NewClient(true)
	sseClient := phttp.NewSSEClient(true)
	return client.NewDirectAPIClient(httpClient, sseClient, fmt.Sprintf("http://%s:%d", host, port), "")
}

func CloudInternal() (cloud.TestKubeCloudAPIClient, error) {
	cloudMu.Lock()
	defer cloudMu.Unlock()

	var err error
	if cloudClient == nil {
		cfg := config2.Config().Worker.Connection
		logger := log.NewSilent()
		// TODO(dejan): now metrics are scrapped on each workflow execution and we get an error when connecting to Control Plane even with publicly trusted certificates.
		// Until a better solution is implemented, TLS verification will be skipped.
		cfg.SkipVerify = true
		cloudConn, err = agentclient.NewVeryInsecureGRPCClientDoNotUseThisClientUnlessYouAreReallySureYouKnowWhatYouAreDoing(context.Background(), cfg.TlsInsecure, cfg.Url, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to connect with Cloud: %w", err)
		}
		cloudClient = cloud.NewTestKubeCloudAPIClient(cloudConn)
	}
	return cloudClient, nil
}

func Cloud() (controlplaneclient.Client, error) {
	cfg := config2.Config()
	grpcClient, err := CloudInternal()
	if err != nil {
		return nil, err
	}
	proContext, err := loadProContext()
	if err != nil {
		return nil, err
	}

	proContextInternal := mapProContext(proContext)

	return controlplaneclient.New(grpcClient, proContextInternal, controlplaneclient.ClientOptions{
		StorageSkipVerify:  true, // FIXME?
		ExecutionID:        cfg.Execution.Id,
		WorkflowName:       cfg.Workflow.Name,
		ParentExecutionIDs: strings.Split(cfg.Execution.ParentIds, "/"),
	}, log.DefaultLogger), nil
}

func mapProContext(context *cloud.ProContextResponse) config3.ProContext {
	cfg := config2.Config()
	cloudStorage := capabilities.Enabled(context.Capabilities, capabilities.CapabilityCloudStorage)

	return config3.ProContext{
		// From the control plane's pro context.
		EnvID:        context.EnvId,
		OrgID:        context.OrgId,
		CloudStorage: cloudStorage,
		OrgSlug:      context.OrgSlug,
		DashboardURI: context.PublicDashboardUrl,

		// From the execution pod's annotations.
		APIKey:      cfg.Worker.Connection.ApiKey,
		URL:         cfg.Worker.Connection.Url,
		TLSInsecure: cfg.Worker.Connection.TlsInsecure,
		SkipVerify:  cfg.Worker.Connection.SkipVerify,
		EnvName:     cfg.Execution.EnvironmentId,
		EnvSlug:     cfg.Execution.EnvironmentId,
		OrgName:     cfg.Execution.OrganizationId,
		Agent: config3.ProContextAgent{
			ID:   cfg.Worker.Connection.AgentID,
			Name: cfg.Worker.Connection.AgentID,
			Environments: []config3.ProContextAgentEnvironment{
				{
					ID:   cfg.Execution.EnvironmentId,
					Slug: cfg.Execution.EnvironmentId,
					Name: cfg.Execution.EnvironmentId,
				},
			},
		},
	}
}
