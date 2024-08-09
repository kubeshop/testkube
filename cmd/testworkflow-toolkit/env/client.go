package env

import (
	"context"
	"fmt"
	"math"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/cache"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/configmap"
	phttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

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
	secretClient := &secret.Client{ClientSet: clientSet, Namespace: Namespace(), Log: log.DefaultLogger}
	configMapClient := &configmap.Client{ClientSet: clientSet, Namespace: Namespace(), Log: log.DefaultLogger}
	inspectorStorages := []imageinspector.Storage{imageinspector.NewMemoryStorage()}
	if Config().Images.InspectorPersistenceEnabled {
		configmapStorage := imageinspector.NewConfigMapStorage(configMapClient, Config().Images.InspectorPersistenceCacheKey, true)
		_ = configmapStorage.CopyTo(context.Background(), inspectorStorages[0].(imageinspector.StorageTransfer))
		inspectorStorages = append(inspectorStorages, configmapStorage)
	}
	return imageinspector.NewInspector(
		Config().System.DefaultRegistry,
		imageinspector.NewCraneFetcher(),
		imageinspector.NewSecretFetcher(secretClient, cache.NewInMemoryCache[*corev1.Secret](), imageinspector.WithSecretCacheTTL(Config().Images.ImageCredentialsCacheTTL)),
		inspectorStorages...,
	)
}

func Testkube() client.Client {
	if UseProxy() {
		return client.NewProxyAPIClient(Kubernetes(), client.NewAPIConfig(Namespace(), config.APIServerName, config.APIServerPort))
	}
	httpClient := phttp.NewClient(true)
	sseClient := phttp.NewSSEClient(true)
	return client.NewDirectAPIClient(httpClient, sseClient, fmt.Sprintf("http://%s:%d", config.APIServerName, config.APIServerPort), "")
}

func ObjectStorageClient() (*minio.Client, error) {
	cfg := Config().ObjectStorage
	opts := minio.GetTLSOptions(cfg.Ssl, cfg.SkipVerify, cfg.CertFile, cfg.KeyFile, cfg.CAFile)
	c := minio.NewClient(cfg.Endpoint, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.Region, cfg.Token, cfg.Bucket, opts...)
	return c, c.Connect()
}

func Cloud(ctx context.Context) (cloudexecutor.Executor, cloud.TestKubeCloudAPIClient) {
	cfg := Config().Cloud
	grpcConn, err := agent.NewGRPCConnection(ctx, cfg.TlsInsecure, cfg.SkipVerify, cfg.Url, "", "", "", log.DefaultLogger)
	if err != nil {
		ui.Fail(fmt.Errorf("failed to connect with Cloud: %w", err))
	}
	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)
	return cloudexecutor.NewCloudGRPCExecutor(grpcClient, grpcConn, cfg.ApiKey, cfg.RunnerId), grpcClient
}
