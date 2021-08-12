package newman

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/kubeshop/kubetest/pkg/process"
	"github.com/kubeshop/kubetest/pkg/tmp"
)

type ExecutionResult struct {
	Error     error
	RawOutput string
	Metadata  ExecutionJSONResult
}

func (r ExecutionResult) Err(err error) ExecutionResult {
	r.Error = err
	return r
}

// Runner struct for newman based runner
type Runner struct {
}

// Run runs particular script content on top of newman binary
func (r *Runner) Run(input io.Reader, params map[string]string) (result ExecutionResult) {
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

	tmpName := tmp.Name()
	out, err := process.Execute("newman", "run", path, "-e", envpath, "--reporters", "cli,json", "--reporter-json-export", tmpName)
	if err != nil {
		return result.Err(err)
	}

	bytes, err := ioutil.ReadFile(tmpName)
	if err != nil {
		return result.Err(err)
	}
	err = json.Unmarshal(bytes, &result.Metadata)
	if err != nil {
		return result.Err(err)
	}

	result.RawOutput = string(out)

	return
}
