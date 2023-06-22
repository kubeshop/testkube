package agent

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
)

// Run starts test runner, test runner can have 3 states
// - pod:success, test execution: success
// - pod:success, test execution: failed
// - pod:failed,  test execution: failed - this one is unusual behaviour
func Run(ctx context.Context, r runner.Runner, args []string) {

	var test []byte
	var err error

	stat, _ := os.Stdin.Stat()
	switch {
	case (stat.Mode() & os.ModeCharDevice) == 0:
		test, err = io.ReadAll(os.Stdin)
		if err != nil {
			output.PrintError(os.Stderr, errors.Errorf("can't read stdin input: %v", err))
			os.Exit(1)
		}
	case len(args) > 1:
		test = []byte(args[1])
		hasFileFlag := args[1] == "-f" || args[1] == "--file"
		if hasFileFlag {
			test, err = os.ReadFile(args[2])
			if err != nil {
				output.PrintError(os.Stderr, errors.Errorf("error reading JSON file: %v", err))
				os.Exit(1)
			}
		}
	default:
		output.PrintError(os.Stderr, errors.Errorf("execution json must be provided using stdin, program argument or -f|--file flag"))
		os.Exit(1)
	}

	e := testkube.Execution{}

	err = json.Unmarshal(test, &e)
	if err != nil {
		output.PrintError(os.Stderr, errors.Wrap(err, "error unmarshalling execution json"))
		os.Exit(1)
	}

	if r.GetType().IsMain() && e.PreRunScript != "" {
		output.PrintEvent("running prerun script", e.Id)

		if serr := runScript(e.PreRunScript); serr != nil {
			output.PrintError(os.Stderr, serr)
			os.Exit(1)
		}
	}

	output.PrintEvent("running test", e.Id)
	result, err := r.Run(ctx, e)

	if r.GetType().IsMain() && e.PostRunScript != "" {
		output.PrintEvent("running postrun script", e.Id)

		if serr := runScript(e.PostRunScript); serr != nil {
			output.PrintError(os.Stderr, serr)
			os.Exit(1)
		}
	}

	if err != nil {
		output.PrintError(os.Stderr, err)
		os.Exit(1)
	}

	output.PrintResult(result)
}

func runScript(body string) error {
	scriptFile, err := os.CreateTemp("", "runscript*.sh")
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
