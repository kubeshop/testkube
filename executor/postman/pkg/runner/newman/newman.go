package newman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/tmp"
)

func NewNewmanRunner() *NewmanRunner {
	return &NewmanRunner{
		Fetcher: content.NewFetcher(""),
	}
}

// NewmanRunner struct for newman based runner
type NewmanRunner struct {
	Fetcher content.ContentFetcher
}

// Run runs particular test content on top of newman binary
func (r *NewmanRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {

	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		return result, err
	}

	if !execution.Content.IsFile() {
		return result, testkube.ErrTestContentTypeNotFile
	}

	// write params to tmp file
	envReader, err := NewEnvFileReader(execution.Variables, execution.VariablesFile, getSecretEnvs())
	if err != nil {
		return result, err
	}
	envpath, err := tmp.ReaderToTmpfile(envReader)
	if err != nil {
		return result, err
	}

	tmpName := tmp.Name() + ".json"

	args := []string{
		"run", path, "-e", envpath, "--reporters", "cli,json", "--reporter-json-export", tmpName,
	}
	args = append(args, execution.Args...)

	// we'll get error here in case of failed test too so we treat this as
	// starter test execution with failed status
	out, err := executor.Run("", "newman", args...)

	// try to get json result even if process returned error (could be invalid test)
	newmanResult, nerr := r.GetNewmanResult(tmpName, out)
	// convert newman result to OpenAPI struct
	result = MapMetadataToResult(newmanResult)

	// catch errors if any
	if err != nil {
		return result.Err(err), nil
	}

	if nerr != nil {
		return result.Err(nerr), nil
	}

	return result, nil
}

func (r NewmanRunner) GetNewmanResult(tmpName string, out []byte) (newmanResult NewmanExecutionResult, err error) {
	newmanResult.Output = string(out)

	// parse JSON output of newman test
	bytes, err := ioutil.ReadFile(tmpName)
	if err != nil {
		return newmanResult, err
	}

	err = json.Unmarshal(bytes, &newmanResult.Metadata)
	if err != nil {
		return newmanResult, err
	}

	return
}

func getSecretEnvs() (secretEnvs []string) {
	i := 1
	for {
		secretEnv := os.Getenv(fmt.Sprintf("RUNNER_SECRET_ENV%d", i))
		if secretEnv == "" {
			break
		}

		secretEnvs = append(secretEnvs, secretEnv)
		i++
	}

	return secretEnvs
}
