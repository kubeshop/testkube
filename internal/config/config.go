package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	APIServerPort                               string        `envconfig:"APISERVER_PORT" default:"8088"`
	APIServerConfig                             string        `envconfig:"APISERVER_CONFIG" default:""`
	APIServerFullname                           string        `envconfig:"APISERVER_FULLNAME" default:"testkube-api-server"`
	APIMongoDSN                                 string        `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	APIMongoAllowTLS                            bool          `envconfig:"API_MONGO_ALLOW_TLS" default:"false"`
	APIMongoSSLCert                             string        `envconfig:"API_MONGO_SSL_CERT" default:""`
	APIMongoSSLCAFileKey                        string        `envconfig:"API_MONGO_SSL_CA_FILE_KEY" default:"sslCertificateAuthorityFile"`
	APIMongoSSLClientFileKey                    string        `envconfig:"API_MONGO_SSL_CLIENT_FILE_KEY" default:"sslClientCertificateKeyFile"`
	APIMongoSSLClientFilePass                   string        `envconfig:"API_MONGO_SSL_CLIENT_FILE_PASS_KEY" default:"sslClientCertificateKeyFilePassword"`
	APIMongoAllowDiskUse                        bool          `envconfig:"API_MONGO_ALLOW_DISK_USE" default:"false"`
	APIMongoDB                                  string        `envconfig:"API_MONGO_DB" default:"testkube"`
	APIMongoDBType                              string        `envconfig:"API_MONGO_DB_TYPE" default:"mongo"`
	SlackToken                                  string        `envconfig:"SLACK_TOKEN" default:""`
	SlackConfig                                 string        `envconfig:"SLACK_CONFIG" default:""`
	SlackTemplate                               string        `envconfig:"SLACK_TEMPLATE" default:""`
	StorageEndpoint                             string        `envconfig:"STORAGE_ENDPOINT" default:"localhost:9000"`
	StorageBucket                               string        `envconfig:"STORAGE_BUCKET" default:"testkube-logs"`
	StorageExpiration                           int           `envconfig:"STORAGE_EXPIRATION"`
	StorageAccessKeyID                          string        `envconfig:"STORAGE_ACCESSKEYID" default:""`
	StorageSecretAccessKey                      string        `envconfig:"STORAGE_SECRETACCESSKEY" default:""`
	StorageRegion                               string        `envconfig:"STORAGE_REGION" default:""`
	StorageToken                                string        `envconfig:"STORAGE_TOKEN" default:""`
	StorageSSL                                  bool          `envconfig:"STORAGE_SSL" default:"false"`
	StorageSkipVerify                           bool          `envconfig:"STORAGE_SKIP_VERIFY" default:"false"`
	StorageCertFile                             string        `envconfig:"STORAGE_CERT_FILE" default:""`
	StorageKeyFile                              string        `envconfig:"STORAGE_KEY_FILE" default:""`
	StorageCAFile                               string        `envconfig:"STORAGE_CA_FILE" default:""`
	ScrapperEnabled                             bool          `envconfig:"SCRAPPERENABLED" default:"false"`
	LogsBucket                                  string        `envconfig:"LOGS_BUCKET" default:""`
	LogsStorage                                 string        `envconfig:"LOGS_STORAGE" default:""`
	NatsURI                                     string        `envconfig:"NATS_URI" default:"nats://localhost:4222"`
	NatsSecure                                  bool          `envconfig:"NATS_SECURE" default:"false"`
	NatsSkipVerify                              bool          `envconfig:"NATS_SKIP_VERIFY" default:"false"`
	NatsCertFile                                string        `envconfig:"NATS_CERT_FILE" default:""`
	NatsKeyFile                                 string        `envconfig:"NATS_KEY_FILE" default:""`
	NatsCAFile                                  string        `envconfig:"NATS_CA_FILE" default:""`
	NatsConnectTimeout                          time.Duration `envconfig:"NATS_CONNECT_TIMEOUT" default:"5s"`
	JobServiceAccountName                       string        `envconfig:"JOB_SERVICE_ACCOUNT_NAME" default:""`
	JobTemplateFile                             string        `envconfig:"JOB_TEMPLATE_FILE" default:""`
	DisableTestTriggers                         bool          `envconfig:"DISABLE_TEST_TRIGGERS" default:"false"`
	TestkubeDefaultExecutors                    string        `envconfig:"TESTKUBE_DEFAULT_EXECUTORS" default:""`
	TestkubeEnabledExecutors                    string        `envconfig:"TESTKUBE_ENABLED_EXECUTORS" default:""`
	TestkubeTemplateJob                         string        `envconfig:"TESTKUBE_TEMPLATE_JOB" default:""`
	TestkubeContainerTemplateJob                string        `envconfig:"TESTKUBE_CONTAINER_TEMPLATE_JOB" default:""`
	TestkubeContainerTemplateScraper            string        `envconfig:"TESTKUBE_CONTAINER_TEMPLATE_SCRAPER" default:""`
	TestkubeContainerTemplatePVC                string        `envconfig:"TESTKUBE_CONTAINER_TEMPLATE_PVC" default:""`
	TestkubeTemplateSlavePod                    string        `envconfig:"TESTKUBE_TEMPLATE_SLAVE_POD" default:""`
	TestkubeConfigDir                           string        `envconfig:"TESTKUBE_CONFIG_DIR" default:"config"`
	TestkubeAnalyticsEnabled                    bool          `envconfig:"TESTKUBE_ANALYTICS_ENABLED" default:"false"`
	TestkubeReadonlyExecutors                   bool          `envconfig:"TESTKUBE_READONLY_EXECUTORS" default:"false"`
	TestkubeNamespace                           string        `envconfig:"TESTKUBE_NAMESPACE" default:"testkube"`
	TestkubeOAuthClientID                       string        `envconfig:"TESTKUBE_OAUTH_CLIENTID" default:""`
	TestkubeOAuthClientSecret                   string        `envconfig:"TESTKUBE_OAUTH_CLIENTSECRET" default:""`
	TestkubeOAuthProvider                       string        `envconfig:"TESTKUBE_OAUTH_PROVIDER" default:""`
	TestkubeOAuthScopes                         string        `envconfig:"TESTKUBE_OAUTH_SCOPES" default:""`
	TestkubeProAPIKey                           string        `envconfig:"TESTKUBE_PRO_API_KEY" default:""`
	TestkubeProURL                              string        `envconfig:"TESTKUBE_PRO_URL" default:""`
	TestkubeProTLSInsecure                      bool          `envconfig:"TESTKUBE_PRO_TLS_INSECURE" default:"false"`
	TestkubeProWorkerCount                      int           `envconfig:"TESTKUBE_PRO_WORKER_COUNT" default:"50"`
	TestkubeProLogStreamWorkerCount             int           `envconfig:"TESTKUBE_PRO_LOG_STREAM_WORKER_COUNT" default:"25"`
	TestkubeProWorkflowNotificationsWorkerCount int           `envconfig:"TESTKUBE_PRO_WORKFLOW_NOTIFICATIONS_STREAM_WORKER_COUNT" default:"25"`
	TestkubeProSkipVerify                       bool          `envconfig:"TESTKUBE_PRO_SKIP_VERIFY" default:"false"`
	TestkubeProEnvID                            string        `envconfig:"TESTKUBE_PRO_ENV_ID" default:""`
	TestkubeProOrgID                            string        `envconfig:"TESTKUBE_PRO_ORG_ID" default:""`
	TestkubeProMigrate                          string        `envconfig:"TESTKUBE_PRO_MIGRATE" default:"false"`
	TestkubeProConnectionTimeout                int           `envconfig:"TESTKUBE_PRO_CONNECTION_TIMEOUT" default:"10"`
	TestkubeProCertFile                         string        `envconfig:"TESTKUBE_PRO_CERT_FILE" default:""`
	TestkubeProKeyFile                          string        `envconfig:"TESTKUBE_PRO_KEY_FILE" default:""`
	TestkubeProCAFile                           string        `envconfig:"TESTKUBE_PRO_CA_FILE" default:""`
	TestkubeProTLSSecret                        string        `envconfig:"TESTKUBE_PRO_TLS_SECRET" default:""`
	TestkubeWatcherNamespaces                   string        `envconfig:"TESTKUBE_WATCHER_NAMESPACES" default:""`
	GraphqlPort                                 string        `envconfig:"TESTKUBE_GRAPHQL_PORT" default:"8070"`
	TestkubeRegistry                            string        `envconfig:"TESTKUBE_REGISTRY" default:""`
	TestkubePodStartTimeout                     time.Duration `envconfig:"TESTKUBE_POD_START_TIMEOUT" default:"30m"`
	CDEventsTarget                              string        `envconfig:"CDEVENTS_TARGET" default:""`
	TestkubeDashboardURI                        string        `envconfig:"TESTKUBE_DASHBOARD_URI" default:""`
	DisableReconciler                           bool          `envconfig:"DISABLE_RECONCILER" default:"false"`
	TestkubeClusterName                         string        `envconfig:"TESTKUBE_CLUSTER_NAME" default:""`
	CompressArtifacts                           bool          `envconfig:"COMPRESSARTIFACTS" default:"false"`
	TestkubeHelmchartVersion                    string        `envconfig:"TESTKUBE_HELMCHART_VERSION" default:""`
	DebugListenAddr                             string        `envconfig:"DEBUG_LISTEN_ADDR" default:"0.0.0.0:1337"`
	EnableDebugServer                           bool          `envconfig:"ENABLE_DEBUG_SERVER" default:"false"`
	EnableSecretsEndpoint                       bool          `envconfig:"ENABLE_SECRETS_ENDPOINT" default:"false"`
	DisableMongoMigrations                      bool          `envconfig:"DISABLE_MONGO_MIGRATIONS" default:"false"`
	Debug                                       bool          `envconfig:"DEBUG" default:"false"`
	EnableImageDataPersistentCache              bool          `envconfig:"TESTKUBE_ENABLE_IMAGE_DATA_PERSISTENT_CACHE" default:"false"`
	ImageDataPersistentCacheKey                 string        `envconfig:"TESTKUBE_IMAGE_DATA_PERSISTENT_CACHE_KEY" default:"testkube-image-cache"`
	LogServerGrpcAddress                        string        `envconfig:"LOG_SERVER_GRPC_ADDRESS" default:":9090"`
	LogServerSecure                             bool          `envconfig:"LOG_SERVER_SECURE" default:"false"`
	LogServerSkipVerify                         bool          `envconfig:"LOG_SERVER_SKIP_VERIFY" default:"false"`
	LogServerCertFile                           string        `envconfig:"LOG_SERVER_CERT_FILE" default:""`
	LogServerKeyFile                            string        `envconfig:"LOG_SERVER_KEY_FILE" default:""`
	LogServerCAFile                             string        `envconfig:"LOG_SERVER_CA_FILE" default:""`
	DisableSecretCreation                       bool          `envconfig:"DISABLE_SECRET_CREATION" default:"false"`
	TestkubeExecutionNamespaces                 string        `envconfig:"TESTKUBE_EXECUTION_NAMESPACES" default:""`

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

func Get() (*Config, error) {
	config := Config{}
	if err := envconfig.Process("config", &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// CleanLegacyVars configures new environment variables from the deprecated ones
func (c *Config) CleanLegacyVars() {
	if c.TestkubeProAPIKey == "" && c.TestkubeCloudAPIKey != "" {
		c.TestkubeProAPIKey = c.TestkubeCloudAPIKey
	}

	if c.TestkubeProURL == "" && c.TestkubeCloudURL != "" {
		c.TestkubeProURL = c.TestkubeCloudURL
	}

	if !c.TestkubeProTLSInsecure && c.TestkubeCloudTLSInsecure {
		c.TestkubeProTLSInsecure = c.TestkubeCloudTLSInsecure
	}

	if c.TestkubeProWorkerCount == 0 && c.TestkubeCloudWorkerCount != 0 {
		c.TestkubeProWorkerCount = c.TestkubeCloudWorkerCount
	}

	if c.TestkubeProLogStreamWorkerCount == 0 && c.TestkubeCloudLogStreamWorkerCount != 0 {
		c.TestkubeProLogStreamWorkerCount = c.TestkubeCloudLogStreamWorkerCount
	}

	if c.TestkubeProEnvID == "" && c.TestkubeCloudEnvID != "" {
		c.TestkubeProEnvID = c.TestkubeCloudEnvID
	}

	if c.TestkubeProOrgID == "" && c.TestkubeCloudOrgID != "" {
		c.TestkubeProOrgID = c.TestkubeCloudOrgID
	}

	if c.TestkubeProMigrate == "" && c.TestkubeCloudMigrate != "" {
		c.TestkubeProMigrate = c.TestkubeCloudMigrate
	}
}
