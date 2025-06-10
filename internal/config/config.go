package config

import (
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type APIConfig struct {
	// Server
	APIServerPort     int    `envconfig:"APISERVER_PORT" default:"8088"`
	APIServerConfig   string `envconfig:"APISERVER_CONFIG" default:""`
	APIServerFullname string `envconfig:"APISERVER_FULLNAME" default:"testkube-api-server"`

	// GraphQL
	GraphqlPort int `envconfig:"TESTKUBE_GRAPHQL_PORT" default:"8070"`
}

type OSSControlPlaneConfig struct {
	// Server
	GRPCServerPort int `envconfig:"GRPCSERVER_PORT" default:"8089"`

	// Mongo
	APIMongoDSN               string `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	APIMongoAllowTLS          bool   `envconfig:"API_MONGO_ALLOW_TLS" default:"false"`
	APIMongoSSLCert           string `envconfig:"API_MONGO_SSL_CERT" default:""`
	APIMongoSSLCAFileKey      string `envconfig:"API_MONGO_SSL_CA_FILE_KEY" default:"sslCertificateAuthorityFile"`
	APIMongoSSLClientFileKey  string `envconfig:"API_MONGO_SSL_CLIENT_FILE_KEY" default:"sslClientCertificateKeyFile"`
	APIMongoSSLClientFilePass string `envconfig:"API_MONGO_SSL_CLIENT_FILE_PASS_KEY" default:"sslClientCertificateKeyFilePassword"`
	APIMongoAllowDiskUse      bool   `envconfig:"API_MONGO_ALLOW_DISK_USE" default:"false"`
	APIMongoDB                string `envconfig:"API_MONGO_DB" default:"testkube"`
	APIMongoDBType            string `envconfig:"API_MONGO_DB_TYPE" default:"mongo"`
	DisableMongoMigrations    bool   `envconfig:"DISABLE_MONGO_MIGRATIONS" default:"false"`

	// Postgres
	APIPostgresDSN string `envconfig:"API_POSTGRES_DSN" default:""`

	// Minio
	StorageEndpoint        string `envconfig:"STORAGE_ENDPOINT" default:"localhost:9000"`
	StorageBucket          string `envconfig:"STORAGE_BUCKET" default:"testkube-logs"`
	StorageExpiration      int    `envconfig:"STORAGE_EXPIRATION"`
	StorageAccessKeyID     string `envconfig:"STORAGE_ACCESSKEYID" default:""`
	StorageSecretAccessKey string `envconfig:"STORAGE_SECRETACCESSKEY" default:""`
	StorageRegion          string `envconfig:"STORAGE_REGION" default:""`
	StorageToken           string `envconfig:"STORAGE_TOKEN" default:""`
	StorageSSL             bool   `envconfig:"STORAGE_SSL" default:"false"`
	StorageSkipVerify      bool   `envconfig:"STORAGE_SKIP_VERIFY" default:"false"`
	StorageCertFile        string `envconfig:"STORAGE_CERT_FILE" default:""`
	StorageKeyFile         string `envconfig:"STORAGE_KEY_FILE" default:""`
	StorageCAFile          string `envconfig:"STORAGE_CA_FILE" default:""`

	LogsBucket  string `envconfig:"LOGS_BUCKET" default:""`
	LogsStorage string `envconfig:"LOGS_STORAGE" default:""`
}

type LegacyExecutorConfig struct {
	// WhitelistedContainers is a list of containers from which logs should be collected.
	WhitelistedContainers            []string      `envconfig:"WHITELISTED_CONTAINERS" default:"init,logs,scraper"`
	ScrapperEnabled                  bool          `envconfig:"SCRAPPERENABLED" default:"false"`
	JobServiceAccountName            string        `envconfig:"JOB_SERVICE_ACCOUNT_NAME" default:""`
	JobTemplateFile                  string        `envconfig:"JOB_TEMPLATE_FILE" default:""`
	DisableTestTriggers              bool          `envconfig:"DISABLE_TEST_TRIGGERS" default:"false"`
	TestkubeDefaultExecutors         string        `envconfig:"TESTKUBE_DEFAULT_EXECUTORS" default:""`
	TestkubeEnabledExecutors         string        `envconfig:"TESTKUBE_ENABLED_EXECUTORS" default:""`
	TestkubeTemplateJob              string        `envconfig:"TESTKUBE_TEMPLATE_JOB" default:""`
	TestkubeContainerTemplateJob     string        `envconfig:"TESTKUBE_CONTAINER_TEMPLATE_JOB" default:""`
	TestkubeContainerTemplateScraper string        `envconfig:"TESTKUBE_CONTAINER_TEMPLATE_SCRAPER" default:""`
	TestkubeContainerTemplatePVC     string        `envconfig:"TESTKUBE_CONTAINER_TEMPLATE_PVC" default:""`
	TestkubeTemplateSlavePod         string        `envconfig:"TESTKUBE_TEMPLATE_SLAVE_POD" default:""`
	TestkubeReadonlyExecutors        bool          `envconfig:"TESTKUBE_READONLY_EXECUTORS" default:"false"`
	CompressArtifacts                bool          `envconfig:"COMPRESSARTIFACTS" default:"false"`
	TestkubeDefaultStorageClassName  string        `envconfig:"TESTKUBE_DEFAULT_STORAGE_CLASS_NAME" default:""`
	TestkubePodStartTimeout          time.Duration `envconfig:"TESTKUBE_POD_START_TIMEOUT" default:"30m"`

	DisableReconciler bool `envconfig:"DISABLE_RECONCILER" default:"false"`
}

type NatsConfig struct {
	NatsEmbedded         bool          `envconfig:"NATS_EMBEDDED" default:"false"`
	NatsEmbeddedStoreDir string        `envconfig:"NATS_EMBEDDED_STORE_DIR" default:"/app/nats"`
	NatsURI              string        `envconfig:"NATS_URI" default:"nats://localhost:4222"`
	NatsSecure           bool          `envconfig:"NATS_SECURE" default:"false"`
	NatsSkipVerify       bool          `envconfig:"NATS_SKIP_VERIFY" default:"false"`
	NatsCertFile         string        `envconfig:"NATS_CERT_FILE" default:""`
	NatsKeyFile          string        `envconfig:"NATS_KEY_FILE" default:""`
	NatsCAFile           string        `envconfig:"NATS_CA_FILE" default:""`
	NatsConnectTimeout   time.Duration `envconfig:"NATS_CONNECT_TIMEOUT" default:"5s"`
}

type KubernetesEventListenerConfig struct {
	TestkubeWatcherNamespaces string `envconfig:"TESTKUBE_WATCHER_NAMESPACES" default:""`
}

type LogServerConfig struct {
	LogServerGrpcAddress string `envconfig:"LOG_SERVER_GRPC_ADDRESS" default:":9090"`
	LogServerSecure      bool   `envconfig:"LOG_SERVER_SECURE" default:"false"`
	LogServerSkipVerify  bool   `envconfig:"LOG_SERVER_SKIP_VERIFY" default:"false"`
	LogServerCertFile    string `envconfig:"LOG_SERVER_CERT_FILE" default:""`
	LogServerKeyFile     string `envconfig:"LOG_SERVER_KEY_FILE" default:""`
	LogServerCAFile      string `envconfig:"LOG_SERVER_CA_FILE" default:""`
}

type ControlPlaneConfig struct {
	TestkubeProEnvID             string        `envconfig:"TESTKUBE_PRO_ENV_ID" default:""`
	TestkubeProOrgID             string        `envconfig:"TESTKUBE_PRO_ORG_ID" default:""`
	TestkubeProAgentID           string        `envconfig:"TESTKUBE_PRO_AGENT_ID" default:""`
	TestkubeProAPIKey            string        `envconfig:"TESTKUBE_PRO_API_KEY" default:""`
	TestkubeProURL               string        `envconfig:"TESTKUBE_PRO_URL" default:""`
	TestkubeProTLSInsecure       bool          `envconfig:"TESTKUBE_PRO_TLS_INSECURE" default:"false"`
	TestkubeProSkipVerify        bool          `envconfig:"TESTKUBE_PRO_SKIP_VERIFY" default:"false"`
	TestkubeProConnectionTimeout int           `envconfig:"TESTKUBE_PRO_CONNECTION_TIMEOUT" default:"10"`
	TestkubeProCertFile          string        `envconfig:"TESTKUBE_PRO_CERT_FILE" default:""`
	TestkubeProKeyFile           string        `envconfig:"TESTKUBE_PRO_KEY_FILE" default:""`
	TestkubeProTLSSecret         string        `envconfig:"TESTKUBE_PRO_TLS_SECRET" default:""`
	TestkubeProSendTimeout       time.Duration `envconfig:"TESTKUBE_PRO_SEND_TIMEOUT" default:"30s"`
	TestkubeProRecvTimeout       time.Duration `envconfig:"TESTKUBE_PRO_RECV_TIMEOUT" default:"5m"`

	// TestkubeProCAFile is meant to provide a custom CA when making a TLS connection to
	// the agent API.
	//
	// Deprecated: Instead mount a CA file into a directory and specify the directory
	// path with the SSL_CERT_DIR environment variable.
	TestkubeProCAFile string `envconfig:"TESTKUBE_PRO_CA_FILE" default:""`
}

type DeprecatedControlPlaneConfig struct {
	// DEPRECATED: Use TestkubeProAPIKey instead
	TestkubeCloudAPIKey string `envconfig:"TESTKUBE_CLOUD_API_KEY" default:""`
	// DEPRECATED: Use TestkubeProURL instead
	TestkubeCloudURL string `envconfig:"TESTKUBE_CLOUD_URL" default:""`
	// DEPRECATED: Use TestkubeProTLSInsecure instead
	TestkubeCloudTLSInsecure bool `envconfig:"TESTKUBE_CLOUD_TLS_INSECURE" default:"false"`
	// DEPRECATED: Use TestkubeProWorkerCount instead
	TestkubeCloudWorkerCount int `envconfig:"TESTKUBE_CLOUD_WORKER_COUNT" default:"50"`
	// DEPRECATED: Use TestkubeProLogStreamWorkerCount instead
	TestkubeCloudLogStreamWorkerCount int `envconfig:"TESTKUBE_CLOUD_LOG_STREAM_WORKER_COUNT" default:"25"`
	// DEPRECATED: Use TestkubeProEnvID instead
	TestkubeCloudEnvID string `envconfig:"TESTKUBE_CLOUD_ENV_ID" default:""`
	// DEPRECATED: Use TestkubeProOrgID instead
	TestkubeCloudOrgID string `envconfig:"TESTKUBE_CLOUD_ORG_ID" default:""`
	// DEPRECATED: Use TestkubeProMigrate instead
	TestkubeCloudMigrate string `envconfig:"TESTKUBE_CLOUD_MIGRATE" default:"false"`
}

type SlackIntegrationConfig struct {
	SlackToken    string `envconfig:"SLACK_TOKEN" default:""`
	SlackConfig   string `envconfig:"SLACK_CONFIG" default:""`
	SlackTemplate string `envconfig:"SLACK_TEMPLATE" default:""`
}

type SecretManagementConfig struct {
	EnableSecretsEndpoint   bool   `envconfig:"ENABLE_SECRETS_ENDPOINT" default:"false"`
	EnableListingAllSecrets bool   `envconfig:"ENABLE_LISTING_ALL_SECRETS" default:"false"`
	SecretCreationPrefix    string `envconfig:"SECRET_CREATION_PREFIX" default:"testkube-"`
}

type ImageInspectorConfig struct {
	TestkubeRegistry string `envconfig:"TESTKUBE_REGISTRY" default:""`
	// TestkubeImageCredentialsCacheTTL is the duration for which the image pull credentials should be cached provided as a Go duration string.
	// If set to 0, the cache is disabled.
	TestkubeImageCredentialsCacheTTL time.Duration `envconfig:"TESTKUBE_IMAGE_CREDENTIALS_CACHE_TTL" default:"30m"`
	EnableImageDataPersistentCache   bool          `envconfig:"TESTKUBE_ENABLE_IMAGE_DATA_PERSISTENT_CACHE" default:"false"`
	ImageDataPersistentCacheKey      string        `envconfig:"TESTKUBE_IMAGE_DATA_PERSISTENT_CACHE_KEY" default:"testkube-image-cache"`
}

type RunnerConfig struct {
	DefaultExecutionNamespace string `envconfig:"DEFAULT_EXECUTION_NAMESPACE" default:""`
	DisableRunner             bool   `envconfig:"DISABLE_RUNNER" default:"false"`
}

type GitOpsSyncConfig struct {
	GitOpsSyncKubernetesToCloudEnabled bool   `envconfig:"GITOPS_KUBERNETES_TO_CLOUD_ENABLED" default:"false"`
	GitOpsSyncCloudToKubernetesEnabled bool   `envconfig:"GITOPS_CLOUD_TO_KUBERNETES_ENABLED" default:"false"`
	GitOpsSyncCloudNamePattern         string `envconfig:"GITOPS_CLOUD_NAME_PATTERN" default:"<name>"`
	GitOpsSyncKubernetesNamePattern    string `envconfig:"GITOPS_KUBERNETES_NAME_PATTERN" default:"<name>"`
}

type CronJobConfig struct {
	EnableCronJobs string `envconfig:"ENABLE_CRON_JOBS" default:""`
}

type Config struct {
	APIConfig
	OSSControlPlaneConfig
	LegacyExecutorConfig
	NatsConfig
	KubernetesEventListenerConfig
	LogServerConfig
	ControlPlaneConfig
	SlackIntegrationConfig
	SecretManagementConfig
	RunnerConfig
	ImageInspectorConfig
	GitOpsSyncConfig
	CronJobConfig
	DisableDefaultAgent             bool     `envconfig:"DISABLE_DEFAULT_AGENT" default:"false"`
	TestkubeConfigDir               string   `envconfig:"TESTKUBE_CONFIG_DIR" default:"config"`
	TestkubeAnalyticsEnabled        bool     `envconfig:"TESTKUBE_ANALYTICS_ENABLED" default:"false"`
	TestkubeNamespace               string   `envconfig:"TESTKUBE_NAMESPACE" default:"testkube"`
	TestkubeProWorkerCount          int      `envconfig:"TESTKUBE_PRO_WORKER_COUNT" default:"50"`
	TestkubeProLogStreamWorkerCount int      `envconfig:"TESTKUBE_PRO_LOG_STREAM_WORKER_COUNT" default:"25"`
	TestkubeProMigrate              string   `envconfig:"TESTKUBE_PRO_MIGRATE" default:"false"`
	TestkubeProRunnerCustomCASecret string   `envconfig:"TESTKUBE_PRO_RUNNER_CUSTOM_CA_SECRET" default:""`
	CDEventsTarget                  string   `envconfig:"CDEVENTS_TARGET" default:""`
	TestkubeDashboardURI            string   `envconfig:"TESTKUBE_DASHBOARD_URI" default:""`
	TestkubeClusterName             string   `envconfig:"TESTKUBE_CLUSTER_NAME" default:""`
	TestkubeHelmchartVersion        string   `envconfig:"TESTKUBE_HELMCHART_VERSION" default:""`
	DebugListenAddr                 string   `envconfig:"DEBUG_LISTEN_ADDR" default:"0.0.0.0:1337"`
	EnableDebugServer               bool     `envconfig:"ENABLE_DEBUG_SERVER" default:"false"`
	Debug                           bool     `envconfig:"DEBUG" default:"false"`
	Trace                           bool     `envconfig:"TRACE" default:"false"`
	DisableSecretCreation           bool     `envconfig:"DISABLE_SECRET_CREATION" default:"false"`
	TestkubeExecutionNamespaces     string   `envconfig:"TESTKUBE_EXECUTION_NAMESPACES" default:""`
	GlobalWorkflowTemplateName      string   `envconfig:"TESTKUBE_GLOBAL_WORKFLOW_TEMPLATE_NAME" default:""`
	GlobalWorkflowTemplateInline    string   `envconfig:"TESTKUBE_GLOBAL_WORKFLOW_TEMPLATE_INLINE" default:""`
	TransferEnvVariables            []string `envconfig:"TRANSFER_ENV_VARS" default:"GRPC_ENFORCE_ALPN_ENABLED"`
	EnableK8sEvents                 bool     `envconfig:"ENABLE_K8S_EVENTS" default:"true"`
	TestkubeDockerImageVersion      string   `envconfig:"TESTKUBE_DOCKER_IMAGE_VERSION" default:""`
	DisableDeprecatedTests          bool     `envconfig:"DISABLE_DEPRECATED_TESTS" default:"false"`
	DisableWebhooks                 bool     `envconfig:"DISABLE_WEBHOOKS" default:"false"`
	AllowLowSecurityFields          bool     `envconfig:"ALLOW_LOW_SECURITY_FIELDS" default:"false"`
	EnableK8sControllers            bool     `envconfig:"ENABLE_K8S_CONTROLLERS" default:"false"`

	FeatureNewArchitecture bool `envconfig:"FEATURE_NEW_ARCHITECTURE" default:"false"`
	FeatureCloudStorage    bool `envconfig:"FEATURE_CLOUD_STORAGE" default:"false"`
}

type DeprecatedConfig struct {
	DeprecatedControlPlaneConfig
}

func Get() (*Config, error) {
	c := Config{}
	if err := envconfig.Process("config", &c); err != nil {
		return nil, err
	}

	deprecated := DeprecatedConfig{}
	if err := envconfig.Process("config", &deprecated); err != nil {
		return nil, err
	}

	if c.TestkubeProAgentID == "" && strings.HasPrefix(c.TestkubeProAPIKey, "tkcagnt_") {
		c.TestkubeProAgentID = strings.Replace(c.TestkubeProEnvID, "tkcenv_", "tkcroot_", 1)
	}

	if strings.HasPrefix(c.TestkubeProAgentID, "tkcrun_") {
		c.DisableTestTriggers = true
		c.DisableWebhooks = true
		c.DisableDeprecatedTests = true
		c.DisableReconciler = true
		c.DisableDefaultAgent = true
		c.NatsEmbedded = true // we don't use it there
		c.EnableCronJobs = "false"
		c.EnableK8sControllers = false
	} else if strings.HasPrefix(c.TestkubeProAgentID, "tkcsync_") {
		c.DisableTestTriggers = true
		c.DisableWebhooks = true
		c.DisableDeprecatedTests = true
		c.DisableReconciler = true
		c.DisableDefaultAgent = true
		c.NatsEmbedded = true // we don't use it there
		c.EnableCronJobs = "false"
		c.EnableK8sControllers = false
	}

	if c.TestkubeProAPIKey == "" && deprecated.TestkubeCloudAPIKey != "" {
		c.TestkubeProAPIKey = deprecated.TestkubeCloudAPIKey
	}
	if c.TestkubeProURL == "" && deprecated.TestkubeCloudURL != "" {
		c.TestkubeProURL = deprecated.TestkubeCloudURL
	}
	if !c.TestkubeProTLSInsecure && deprecated.TestkubeCloudTLSInsecure {
		c.TestkubeProTLSInsecure = deprecated.TestkubeCloudTLSInsecure
	}
	if c.TestkubeProWorkerCount == 0 && deprecated.TestkubeCloudWorkerCount != 0 {
		c.TestkubeProWorkerCount = deprecated.TestkubeCloudWorkerCount
	}
	if c.TestkubeProLogStreamWorkerCount == 0 && deprecated.TestkubeCloudLogStreamWorkerCount != 0 {
		c.TestkubeProLogStreamWorkerCount = deprecated.TestkubeCloudLogStreamWorkerCount
	}
	if c.TestkubeProEnvID == "" && deprecated.TestkubeCloudEnvID != "" {
		c.TestkubeProEnvID = deprecated.TestkubeCloudEnvID
	}
	if c.TestkubeProOrgID == "" && deprecated.TestkubeCloudOrgID != "" {
		c.TestkubeProOrgID = deprecated.TestkubeCloudOrgID
	}
	if c.TestkubeProMigrate == "" && deprecated.TestkubeCloudMigrate != "" {
		c.TestkubeProMigrate = deprecated.TestkubeCloudMigrate
	}

	return &c, nil
}
