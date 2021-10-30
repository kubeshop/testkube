package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	var script []byte
	var err error

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		script, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			output.PrintError(fmt.Errorf("can't read stind input: %w", err))
			os.Exit(1)
		}
	} else if len(args) > 1 {
		script = []byte(args[1])
	} else {
		output.PrintError(fmt.Errorf("missing input JSON argument or stdin input"))
		os.Exit(1)
	}

	output.PrintEvent("running postman/collection from testkube.Execution", string(script))
	e := testkube.Execution{}

	err = json.Unmarshal(script, &e)
	if err != nil {
		output.PrintError(err)
		os.Exit(1)
	}

	result, err := r.Run(e)
	if err != nil {
		output.PrintError(err)
		os.Exit(1)
	}

	output.PrintResult(result)
}
