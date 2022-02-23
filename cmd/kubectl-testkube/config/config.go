package config

import (
	"github.com/kubeshop/testkube/pkg/ui"
)

var storage Storage

// init load default configs for testkube global configuration used in RootCmd
func init() {
	storage = Storage{}
	err := storage.Init()
	ui.WarnOnError("can't init configuration, using default values", err)
}

func Load() (Data, error) {
	return storage.Load()
}

func Save(data Data) error {
	return storage.Save(data)
}
