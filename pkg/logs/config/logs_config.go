package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	NatsURI            string        `envconfig:"NATS_URI" default:"nats://localhost:4222"`
	NatsSecure         bool          `envconfig:"NATS_SECURE" default:"false"`
	NatsSkipVerify     bool          `envconfig:"NATS_SKIP_VERIFY" default:"false"`
	NatsCertFile       string        `envconfig:"NATS_CERT_FILE" default:""`
	NatsKeyFile        string        `envconfig:"NATS_KEY_FILE" default:""`
	NatsCAFile         string        `envconfig:"NATS_CA_FILE" default:""`
	NatsConnectTimeout time.Duration `envconfig:"NATS_CONNECT_TIMEOUT" default:"5s"`
	Namespace          string        `envconfig:"NAMESPACE" default:"testkube"`
	ExecutionId        string        `envconfig:"ID" default:""`
	HttpAddress        string        `envconfig:"HTTP_ADDRESS" default:":8080"`
	GrpcAddress        string        `envconfig:"GRPC_ADDRESS" default:":9090"`
	KVBucketName       string        `envconfig:"KV_BUCKET_NAME" default:"logsState"`
}

func Get() (*Config, error) {
	var config = Config{}
	err := envconfig.Process("config", &config)

	return &config, err
}
