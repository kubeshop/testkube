package featureflags

type FeatureFlags struct {
	LogsV2 bool `envconfig:"FF_LOGS_V2" default:"false"`
}
