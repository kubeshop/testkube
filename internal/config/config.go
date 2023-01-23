package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	APIServerPort             string `envconfig:"APISERVER_PORT" default:"8088"`
	APIServerConfig           string `envconfig:"APISERVER_CONFIG" default:""`
	JobServiceAccountName     string `envconfig:"JOB_SERVICE_ACCOUNT_NAME" default:""`
	LogsStorage               string `envconfig:"LOGS_STORAGE" default:""`
	LogsBucket                string `envconfig:"LOGS_BUCKET" default:""`
	TestkubeAnalyticsEnabled  bool   `envconfig:"TESTKUBE_ANALYTICS_ENABLED" default:"false"`
	TestkubeReadonlyExecutors bool   `envconfig:"TESTKUBE_READONLY_EXECUTORS" default:"false"`
	TestkubeDefaultExecutors  string `envconfig:"TESTKUBE_DEFAULT_EXECUTORS" default:""`
	TestkubeContainerTemplate string `envconfig:"TESTKUBE_CONTAINER_TEMPLATE" default:""`
	TestkubeNamespace         string `envconfig:"TESTKUBE_NAMESPACE" default:"testkube"`
	TestkubeCloudAPIKey       string `envconfig:"TESTKUBE_CLOUD_API_KEY" default:""`
	TestkubeCloudURL          string `envconfig:"TESTKUBE_CLOUD_URL" default:""`
	TestkubeCloudTLSInsecure  bool   `envconfig:"TESTKUBE_CLOUD_TLS_INSECURE" default:"false"`
}

func Get() (*Config, error) {
	config := Config{}
	if err := envconfig.Process("config", &config); err != nil {
		return nil, err
	}
	return &config, nil
}
