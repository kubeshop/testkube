package envs

import (
	"os"
	"strconv"

	"github.com/kubeshop/testkube/pkg/log"
)

func IsTrue(name string) (is bool) {
	var err error
	if val, ok := os.LookupEnv(name); ok {
		is, err = strconv.ParseBool(val)
		if err != nil {
			log.DefaultLogger.Debug("Can't parse bool value for variable %q", name)
		}
	}

	return is
}
