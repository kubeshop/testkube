package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug bool `envconfig:"DEBUG" default:"false"`

	TestkubeProAPIKey               string `envconfig:"TESTKUBE_PRO_API_KEY" default:""`
	TestkubeProURL                  string `envconfig:"TESTKUBE_PRO_URL" default:""`
	TestkubeProLogsPath             string `envconfig:"TESTKUBE_PRO_LOGS_PATH" default:"/logs"`
	TestkubeProTLSInsecure          bool   `envconfig:"TESTKUBE_PRO_TLS_INSECURE" default:"false"`
	TestkubeProWorkerCount          int    `envconfig:"TESTKUBE_PRO_WORKER_COUNT" default:"50"`
	TestkubeProLogStreamWorkerCount int    `envconfig:"TESTKUBE_PRO_LOG_STREAM_WORKER_COUNT" default:"25"`
	TestkubeProSkipVerify           bool   `envconfig:"TESTKUBE_PRO_SKIP_VERIFY" default:"false"`

	NatsURI            string        `envconfig:"NATS_URI" default:"nats://localhost:4222"`
	NatsSecure         bool          `envconfig:"NATS_SECURE" default:"false"`
	NatsSkipVerify     bool          `envconfig:"NATS_SKIP_VERIFY" default:"false"`
	NatsCertFile       string        `envconfig:"NATS_CERT_FILE" default:""`
	NatsKeyFile        string        `envconfig:"NATS_KEY_FILE" default:""`
	NatsCAFile         string        `envconfig:"NATS_CA_FILE" default:""`
	NatsConnectTimeout time.Duration `envconfig:"NATS_CONNECT_TIMEOUT" default:"5s"`

	Namespace        string `envconfig:"NAMESPACE" default:"testkube"`
	ExecutionId      string `envconfig:"ID" default:""`
	HttpAddress      string `envconfig:"HTTP_ADDRESS" default:":8080"`
	GrpcAddress      string `envconfig:"GRPC_ADDRESS" default:":9090"`
	KVBucketName     string `envconfig:"KV_BUCKET_NAME" default:"logsState"`
	TLSSecretName    string `envconfig:"TLS_SECRET_NAME" default:""`
	TLSCertSecretKey string `envconfig:"TLS_CERT_SECRET_KEY" default:""`
	TLSKeySecretKey  string `envconfig:"TLS_KEY_SECRET_KEY" default:""`

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
}

func Get() (*Config, error) {
	var config = Config{}
	err := envconfig.Process("config", &config)

	return &config, err
}
