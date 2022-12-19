package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
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
		test, err = io.ReadAll(os.Stdin)
		if err != nil {
			output.PrintError(os.Stderr, fmt.Errorf("can't read stdin input: %w", err))
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

	if r.GetType().IsMain() && e.PreRunScript != "" {
		output.PrintEvent("running script", e.Id)

		if err = runScript(e.PreRunScript); err != nil {
			output.PrintError(os.Stderr, err)
			os.Exit(1)
		}
	}

	output.PrintEvent("running test", e.Id)
	result, err := r.Run(e)
	if err != nil {
		output.PrintError(os.Stderr, err)
		os.Exit(1)
	}

	output.PrintResult(result)
}

func runScript(body string) error {
	scriptFile, err := os.CreateTemp("", "prerun*.sh")
	if err != nil {
		return err
	}

	filename := scriptFile.Name()
	if _, err = io.Copy(scriptFile, strings.NewReader(body)); err != nil {
		return err
	}

	if err = scriptFile.Close(); err != nil {
		return err
	}

	if err = os.Chmod(filename, 0777); err != nil {
		return err
	}

	if _, err = executor.Run("", "/bin/sh", nil, filename); err != nil {
		return err
	}

	return nil
}
