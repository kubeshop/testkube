package envs

import (
	"os"
	"strconv"
)

func IsTrue(name string) (is bool) {
	if val, ok := os.LookupEnv(name); ok {
		is, _ = strconv.ParseBool(val)
	}

	return is
}
