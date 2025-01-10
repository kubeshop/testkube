package env

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	config2 "github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cache"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/configmap"
	phttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	capabilitiesMu         sync.Mutex
	proContext             *cloud.ProContextResponse
	proContextLoaded       bool
	isNewExecutionsCache   *bool
	isExternalStorageCache *bool
)

func loadCapabilities() {
	capabilitiesMu.Lock()
	defer capabilitiesMu.Unlock()

	// Block if the instance doesn't support that
	cfg := config2.Config()
	if isNewExecutionsCache == nil && cfg.Worker.FeatureFlags[testworkflowconfig.FeatureFlagNewExecutions] != "true" {
		isNewExecutionsCache = common.Ptr(false)
	}
	if isExternalStorageCache == nil && cfg.Worker.FeatureFlags[testworkflowconfig.FeatureFlagTestWorkflowCloudStorage] != "true" {
		isExternalStorageCache = common.Ptr(false)
	}

	// Do not check Cloud support if its already predefined
	if isNewExecutionsCache != nil && isExternalStorageCache != nil {
		return
	}

	// Check support in the cloud
	ctx := agentclient.AddAPIKeyMeta(context.Background(), cfg.Worker.Connection.ApiKey)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !proContextLoaded {
		_, client := Cloud(ctx)
		proContext, _ = client.GetProContext(ctx, &emptypb.Empty{})
		proContextLoaded = true
	}
	if proContext != nil {
		if isNewExecutionsCache == nil {
			isNewExecutionsCache = common.Ptr(capabilities.Enabled(proContext.Capabilities, capabilities.CapabilityNewExecutions))
		}
		if isExternalStorageCache == nil {
			isExternalStorageCache = common.Ptr(capabilities.Enabled(proContext.Capabilities, capabilities.CapabilityTestWorkflowStorage))
		}
	} else {
		isNewExecutionsCache = common.Ptr(false)
		isExternalStorageCache = common.Ptr(false)
	}
}

func IsNewExecutions() bool {
	loadCapabilities()
	return *isNewExecutionsCache
}

func IsExternalStorage() bool {
	loadCapabilities()
	return *isExternalStorageCache
}

func GetCapabilities() []*cloud.Capability {
	loadCapabilities()
	if proContext == nil {
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
		imageinspector.NewCraneFetcher(),
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

var (
	cloudMu       sync.Mutex
	cloudExecutor cloudexecutor.Executor
	cloudClient   cloud.TestKubeCloudAPIClient
	cloudConn     *grpc.ClientConn
)

func Cloud(ctx context.Context) (cloudexecutor.Executor, cloud.TestKubeCloudAPIClient) {
	cloudMu.Lock()
	defer cloudMu.Unlock()

	var err error
	if cloudExecutor == nil {
		cfg := config2.Config().Worker.Connection
		logger := log.NewSilent()
		cloudConn, err = agentclient.NewGRPCConnection(ctx, cfg.TlsInsecure, cfg.SkipVerify, cfg.Url, "", "", "", logger)
		if err != nil {
			ui.Fail(fmt.Errorf("failed to connect with Cloud: %w", err))
		}
		cloudClient = cloud.NewTestKubeCloudAPIClient(cloudConn)
		cloudExecutor = cloudexecutor.NewCloudGRPCExecutor(cloudClient, cfg.ApiKey)
	}

	return cloudExecutor, cloudClient
}
