package executor

import (
	"fmt"
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Run runs executor process wrapped in json line output
// wraps stdout lines into JSON chunks we want it to have common interface for agent
// stdin <- testkube.Execution, stdout <- stream of json logs
// LoggedExecuteInDir will put wrapped JSON output to stdout AND get RAW output into out var
// json logs can be processed later on watch of pod logs
func Run(dir string, command string, envMngr secret.Manager, arguments ...string) (out []byte, err error) {
	obfuscatedArgs := []byte{}
	if envMngr != nil {
		obfuscatedArgs = envMngr.Obfuscate([]byte(strings.Join(arguments, " ")))
	}
	output.PrintLog(fmt.Sprintf("%s Executing in directory %s: \n $ %s %s", ui.IconMicroscope, dir, command, obfuscatedArgs))
	out, err = process.LoggedExecuteInDir(dir, output.NewJSONWrapWriter(os.Stdout, envMngr), command, arguments...)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Execution failed: %s", ui.IconCross, err.Error()))
		return out, err
	}
	output.PrintLog(fmt.Sprintf("%s Execution succeeded", ui.IconCheckMark))
	return out, nil
}
