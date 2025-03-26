package kubernetesworker

import (
	"time"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type NamespaceConfig struct {
	DefaultServiceAccountName string
}

type ClusterConfig struct {
	Id               string
	DefaultNamespace string
	DefaultRegistry  string
	Namespaces       map[string]NamespaceConfig
}

type ImageInspectorConfig struct {
	CacheEnabled bool
	CacheKey     string
	CacheTTL     time.Duration
}

type Config struct {
	Cluster        ClusterConfig
	ImageInspector ImageInspectorConfig
	Connection     testworkflowconfig.WorkerConnectionConfig
	FeatureFlags   map[string]string
	RunnerId       string
}
