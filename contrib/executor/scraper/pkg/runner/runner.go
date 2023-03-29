package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"

	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
)

// NewRunner creates scraper runner
func NewRunner() (*ScraperRunner, error) {
	var params envs.Params
	err := envconfig.Process("runner", &params)
	if err != nil {
		return nil, err
	}

	runner := &ScraperRunner{
		Params: params,
	}

	return runner, nil
}

// ScraperRunner prepares data for executor
type ScraperRunner struct {
	Params envs.Params
}

// Run prepares data for executor
func (r *ScraperRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	// check that the artifact dir exists
	if execution.ArtifactRequest == nil {
		return *result.Err(fmt.Errorf("executor only support artifact based tests")), nil
	}

	_, err = os.Stat(execution.ArtifactRequest.VolumeMountPath)
	if errors.Is(err, os.ErrNotExist) {
		return result, err
	}

	if r.Params.ScrapperEnabled {
		directories := execution.ArtifactRequest.Dirs
		if len(directories) == 0 {
			directories = []string{"."}
		}

		for i := range directories {
			directories[i] = filepath.Join(execution.ArtifactRequest.VolumeMountPath, directories[i])
		}

		output.PrintLog(fmt.Sprintf("Scraping directories: %v", directories))

		if err := factory.Scrape(context.Background(), directories, execution, r.Params); err != nil {
			return result, errors.Wrap(err, "error getting artifacts from SoapUI executor")
		}
		if err != nil {
			return *result.Err(err), fmt.Errorf("failed getting artifacts: %w", err)
		}
	}

	return result, nil
}

// GetType returns runner type
func (r *ScraperRunner) GetType() runner.Type {
	return runner.TypeFin
}
