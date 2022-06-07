package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
)

// Run starts test runner, test runner can have 3 states
// - pod:success, test execution: success
// - pod:success, test execution: failed
// - pod:failed,  test execution: failed - this one is unusual behaviour
func Run(r runner.Runner, args []string) {

	var test []byte
	var err error

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		test, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			output.PrintError(os.Stderr, fmt.Errorf("can't read stind input: %w", err))
			os.Exit(1)
		}
	} else if len(args) > 1 {
		test = []byte(args[1])
	} else {
		output.PrintError(os.Stderr, fmt.Errorf("missing input JSON argument or stdin input"))
		os.Exit(1)
	}

	e := testkube.Execution{}

	err = json.Unmarshal(test, &e)
	if err != nil {
		output.PrintError(os.Stderr, err)
		os.Exit(1)
	}

	output.PrintEvent("running test", e.Id)

	result, err := r.Run(e)
	if err != nil {
		output.PrintError(os.Stderr, err)
		os.Exit(1)
	}

	output.PrintResult(result)
}
