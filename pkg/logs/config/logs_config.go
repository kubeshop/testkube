package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/internal/config"
)

type Config struct {
	Debug bool `envconfig:"DEBUG" default:"false"`

	// Debug variables
	AttachDebugAdapter bool `envconfig:"ATTACH_DEBUG_ADAPTER" default:"false"`

	TestkubeProAPIKey               string `envconfig:"TESTKUBE_PRO_API_KEY" default:""`
	TestkubeProURL                  string `envconfig:"TESTKUBE_PRO_URL" default:""`
	TestkubeProTLSInsecure          bool   `envconfig:"TESTKUBE_PRO_TLS_INSECURE" default:"false"`
	TestkubeProCertFile             string `envconfig:"TESTKUBE_PRO_CERT_FILE" default:""`
	TestkubeProKeyFile              string `envconfig:"TESTKUBE_PRO_KEY_FILE" default:""`
	TestkubeProCAFile               string `envconfig:"TESTKUBE_PRO_CA_FILE" default:""`
	TestkubeProWorkerCount          int    `envconfig:"TESTKUBE_PRO_WORKER_COUNT" default:"50"`
	TestkubeProLogStreamWorkerCount int    `envconfig:"TESTKUBE_PRO_LOG_STREAM_WORKER_COUNT" default:"25"`
	TestkubeProSkipVerify           bool   `envconfig:"TESTKUBE_PRO_SKIP_VERIFY" default:"false"`

	// gRPC Keepalive configuration
	TestkubeProKeepaliveEnabled             bool          `envconfig:"TESTKUBE_PRO_KEEPALIVE_ENABLED" default:"true"`
	TestkubeProKeepaliveTime                time.Duration `envconfig:"TESTKUBE_PRO_KEEPALIVE_TIME" default:"10s"`
	TestkubeProKeepaliveTimeout             time.Duration `envconfig:"TESTKUBE_PRO_KEEPALIVE_TIMEOUT" default:"5s"`
	TestkubeProKeepalivePermitWithoutStream bool          `envconfig:"TESTKUBE_PRO_KEEPALIVE_PERMIT_WITHOUT_STREAM" default:"true"`

	NatsURI            string        `envconfig:"NATS_URI" default:"nats://localhost:4222"`
	NatsSecure         bool          `envconfig:"NATS_SECURE" default:"false"`
	NatsSkipVerify     bool          `envconfig:"NATS_SKIP_VERIFY" default:"false"`
	NatsCertFile       string        `envconfig:"NATS_CERT_FILE" default:""`
	NatsKeyFile        string        `envconfig:"NATS_KEY_FILE" default:""`
	NatsCAFile         string        `envconfig:"NATS_CA_FILE" default:""`
	NatsConnectTimeout time.Duration `envconfig:"NATS_CONNECT_TIMEOUT" default:"5s"`

	Namespace    string `envconfig:"NAMESPACE" default:"testkube"`
	ExecutionId  string `envconfig:"ID" default:""`
	PodName      string `envconfig:"POD_NAME" default:""`
	Source       string `envconfig:"Source" default:""`
	HttpAddress  string `envconfig:"HTTP_ADDRESS" default:":8080"`
	GrpcAddress  string `envconfig:"GRPC_ADDRESS" default:":9090"`
	KVBucketName string `envconfig:"KV_BUCKET_NAME" default:"logsState"`

	GrpcSecure       bool   `envconfig:"GRPC_SECURE" default:"false"`
	GrpcClientAuth   bool   `envconfig:"GRPC_CLIENT_AUTH" default:"false"`
	GrpcCertFile     string `envconfig:"GRPC_CERT_FILE" default:""`
	GrpcKeyFile      string `envconfig:"GRPC_KEY_FILE" default:""`
	GrpcClientCAFile string `envconfig:"GRPC_CLIENT_CA_FILE" default:""`

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
	StorageFilePath        string `envconfig:"STORAGE_FILE_PATH" default:""`
}

func Get() (*Config, error) {
	var config = Config{}
	err := envconfig.Process("config", &config)

	return &config, err
}

// GetKeepaliveConfig creates a KeepaliveConfig from the logs configuration
func (c *Config) GetKeepaliveConfig() config.KeepaliveConfig {
	return config.KeepaliveConfig{
		Enabled:             c.TestkubeProKeepaliveEnabled,
		Time:                c.TestkubeProKeepaliveTime,
		Timeout:             c.TestkubeProKeepaliveTimeout,
		PermitWithoutStream: c.TestkubeProKeepalivePermitWithoutStream,
	}
}
