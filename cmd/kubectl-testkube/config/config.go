package config

import (
	"github.com/kubeshop/testkube/pkg/log"
)

var Config config

// init load default configs for testkube global configuration used in RootCmd
func init() {
	Config = config{}
	l := log.DefaultLogger

	// set default analytics enabled
	Config.Data.AnalyticsEnabled = true

	err := Config.Init(Config.Data)
	if err != nil {
		l.Errorw("can't init configuration", "error", err.Error())
		return
	}
	Config.Data, err = Config.Load()
	if err != nil {
		l.Errorw("can't load configuration file", "error", err.Error())
	}
}

// config is struct for managing state of Config in Storage
type config struct {
	Data
	Storage
}
