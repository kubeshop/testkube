package executor

import (
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Run runs executor process wrapped in json line output
// wraps stdout lines into JSON chunks we want it to have common interface for agent
// stdin <- testkube.Execution, stdout <- stream of json logs
// LoggedExecuteInDir will put wrapped JSON output to stdout AND get RAW output into out var
// json logs can be processed later on watch of pod logs
func Run(dir string, command string, envMngr env.Interface, arguments ...string) (out []byte, err error) {
	var obfuscatedArgs []byte
	if envMngr != nil {
		obfuscatedArgs = envMngr.ObfuscateSecrets([]byte(strings.Join(arguments, " ")))
	}
	output.PrintLogf("%s Executing in directory %s: \n $ %s %s", ui.IconMicroscope, dir, command, obfuscatedArgs)
	out, err = process.LoggedExecuteInDir(dir, output.NewJSONWrapWriter(os.Stdout, envMngr), command, arguments...)
	if err != nil {
		output.PrintLogf("%s Execution failed: %s", ui.IconCross, err.Error())
		return out, err
	}
	output.PrintLogf("%s Execution succeeded", ui.IconCheckMark)
	return out, nil
}

// MergeCommandAndArgs prepares command and args for Run method
func MergeCommandAndArgs(command, arguments []string) (string, []string) {
	cmd := ""
	if len(command) > 0 {
		cmd = command[0]
		arguments = append(command[1:], arguments...)
	}

	return cmd, arguments
}
