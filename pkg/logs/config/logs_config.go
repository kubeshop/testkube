package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	NatsURI     string `envconfig:"NATS_URI" default:"nats://localhost:4222"`
	Namespace   string `envconfig:"NAMESPACE" default:"testkube"`
	ExecutionId string `envconfig:"ID" default:""`
}

func Get() (*Config, error) {
	var config = Config{}
	err := envconfig.Process("config", &config)

	return &config, err
}
