package services

import (
	"fmt"

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
	serviceAccountNames map[string]string,
	processor testworkflowprocessor.Processor,
) executionworkertypes.Worker {
	namespacesConfig := map[string]kubernetesworker.NamespaceConfig{}
	for n, s := range serviceAccountNames {
		namespacesConfig[n] = kubernetesworker.NamespaceConfig{DefaultServiceAccountName: s}
	}
	cloudUrl := cfg.TestkubeProURL
	cloudApiKey := cfg.TestkubeProAPIKey
	objectStorageConfig := testworkflowconfig.ObjectStorageConfig{}
	if cloudApiKey == "" {
		cloudUrl = ""
		objectStorageConfig = testworkflowconfig.ObjectStorageConfig{
			Endpoint:        cfg.StorageEndpoint,
			AccessKeyID:     cfg.StorageAccessKeyID,
			SecretAccessKey: cfg.StorageSecretAccessKey,
			Region:          cfg.StorageRegion,
			Token:           cfg.StorageToken,
			Bucket:          cfg.StorageBucket,
			Ssl:             cfg.StorageSSL,
			SkipVerify:      cfg.StorageSkipVerify,
			CertFile:        cfg.StorageCertFile,
			KeyFile:         cfg.StorageKeyFile,
			CAFile:          cfg.StorageCAFile,
		}
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
			Url:         cloudUrl,
			ApiKey:      cloudApiKey,
			SkipVerify:  cfg.TestkubeProSkipVerify,
			TlsInsecure: cfg.TestkubeProTLSInsecure,

			// TODO: Prepare ControlPlane interface for OSS, so we may unify the communication
			LocalApiUrl:   fmt.Sprintf("http://%s:%s", cfg.APIServerFullname, cfg.APIServerPort),
			ObjectStorage: objectStorageConfig,
		},
	})
}
