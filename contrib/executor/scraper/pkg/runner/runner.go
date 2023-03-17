package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
)

// NewRunner creates scraper runner
func NewRunner() (*ScraperRunner, error) {
	var params envs.Params
	err := envconfig.Process("runner", &params)
	if err != nil {
		return nil, err
	}

	runner := &ScraperRunner{
		Scraper: scraper.NewMinioScraper(
			params.Endpoint,
			params.AccessKeyID,
			params.SecretAccessKey,
			params.Location,
			params.Token,
			params.Bucket,
			params.Ssl,
		),
		ScrapperEnabled: params.ScrapperEnabled,
	}

	return runner, nil
}

// ScraperRunner prepares data for executor
type ScraperRunner struct {
	ScrapperEnabled bool // RUNNER_SCRAPPERENABLED
	Scraper         scraper.Scraper
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

	if r.ScrapperEnabled {
		directories := execution.ArtifactRequest.Dirs
		if len(directories) == 0 {
			directories = []string{"."}
		}

		for i := range directories {
			directories[i] = filepath.Join(execution.ArtifactRequest.VolumeMountPath, directories[i])
		}

		output.PrintEvent("scraping for test files", directories)
		err := r.Scraper.Scrape(execution.Id, directories)
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
