package runner

import (
	"encoding/json"
	"fmt"

	kubepug "github.com/rikatz/kubepug/pkg/results"
)

// GetResults parses the output of a kubepug execution into a Result
func GetResult(r string) (kubepug.Result, error) {
	var result kubepug.Result
	err := json.Unmarshal([]byte(r), &result)
	if err != nil {
		return result, fmt.Errorf("could not unmarshal result %s: %w", r, err)
	}
	return result, nil
}
