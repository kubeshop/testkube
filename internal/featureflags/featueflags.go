package featureflags

import (
	"github.com/kelseyhightower/envconfig"
)

type FeatureFlags struct {
	LogsV2 bool `envconfig:"FF_LOGS_V2" default:"false"`
}

func Get() (ff FeatureFlags, err error) {
	if err := envconfig.Process("", &ff); err != nil {
		return ff, err
	}
	return
}
