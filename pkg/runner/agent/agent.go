package agent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/runner/output"
)

// Run starts test runner, test runner can have 3 states
// - pod:success, test execution: success
// - pod:success, test execution: failed
// - pod:failed,  test execution: failed - this one is unusual behaviour
func Run(r runner.Runner, args []string) {
	if len(args) == 1 {
		output.NewOutputError(fmt.Errorf("missing input argument"))
		os.Exit(1)
	}

	script := []byte(args[1])

	e := testkube.Execution{}
	json.Unmarshal(script, &e)

	result, err := r.Run(e)
	if err != nil {
		output.PrintError(err)
		os.Exit(1)
	}

	output.PrintResult(result)
}
