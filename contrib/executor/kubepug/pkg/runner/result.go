package runner

import (
	"encoding/json"

	"github.com/pkg/errors"

	kubepug "github.com/rikatz/kubepug/pkg/results"
)

// GetResult parses the output of a kubepug execution into a Result
func GetResult(r string) (kubepug.Result, error) {
	var result kubepug.Result
	err := json.Unmarshal([]byte(r), &result)
	if err != nil {
		return result, errors.Errorf("could not unmarshal result %s: %v", r, err)
	}
	return result, nil
}
