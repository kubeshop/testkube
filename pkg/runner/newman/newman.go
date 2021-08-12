package newman

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/kubeshop/kubetest/pkg/process"
	"github.com/kubeshop/kubetest/pkg/tmp"
)

// Runner struct for newman based runner
type Runner struct {
}

// Run runs particular script content on top of newman binary
func (r *Runner) Run(input io.Reader, params map[string]string) (result kubetest.ExecutionResult) {
	path, err := tmp.ReaderToTmpfile(input)
	if err != nil {
		return result.Err(err)
	}

	// write params to tmp file
	envReader, err := NewEnvFileReader(params)
	if err != nil {
		return result.Err(err)
	}
	envpath, err := tmp.ReaderToTmpfile(envReader)
	if err != nil {
		return result.Err(err)
	}

	var newmanResult NewmanExecutionResult

	tmpName := tmp.Name()
	out, err := process.Execute("newman", "run", path, "-e", envpath, "--reporters", "cli,json", "--reporter-json-export", tmpName)
	if err != nil {
		return result.Err(err)
	}

	newmanResult.RawOutput = string(out)

	// parse JSON output of newman script
	bytes, err := ioutil.ReadFile(tmpName)
	if err != nil {
		return result.Err(err)
	}

	err = json.Unmarshal(bytes, &newmanResult.Metadata)
	if err != nil {
		return result.Err(err)
	}

	// convert newman result to OpenAPI struct
	return MapMetadataToResult(newmanResult)
}
