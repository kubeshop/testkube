package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	NatsURI      string `envconfig:"NATS_URI" default:"nats://localhost:4222"`
	Namespace    string `envconfig:"NAMESPACE" default:"testkube"`
	ExecutionId  string `envconfig:"ID" default:""`
	HttpAddress  string `envconfig:"HTTP_ADDRESS" default:":8080"`
	GrpcAddress  string `envconfig:"GRPC_ADDRESS" default:":9090"`
	KVBucketName string `envconfig:"KV_BUCKET_NAME" default:"logsState"`
}

func Get() (*Config, error) {
	var config = Config{}
	err := envconfig.Process("config", &config)

	return &config, err
}
