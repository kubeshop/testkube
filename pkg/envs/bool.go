package envs

import (
	"os"
	"strconv"
)

func IsTrue(name string) (is bool) {
	var err error
	if val, ok := os.LookupEnv(name); ok {
		is, err = strconv.ParseBool(val)
		if err != nil {
			return false
		}
	}

	return is
}
