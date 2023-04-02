package runner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/storage/minio"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewRunner creates init runner
func NewRunner() *InitRunner {
	dir := os.Getenv("RUNNER_DATADIR")
	return &InitRunner{
		Fetcher: content.NewFetcher(dir),
		dir:     dir,
	}
}

// InitRunner prepares data for executor
type InitRunner struct {
	Fetcher content.ContentFetcher
	dir     string
}

var _ runner.Runner = &InitRunner{}

// Run prepares data for executor
func (r *InitRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLogf("%s Initializing...", ui.IconTruck)
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintLogf("%s Environment variables read unsuccessfully", ui.IconCross)
		return result, errors.Errorf("%s Could not read environment variables: %v", ui.IconCross, err)
	}

	gitUsername := params.GitUsername
	gitToken := params.GitToken

	if gitUsername != "" || gitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = gitUsername
			execution.Content.Repository.Token = gitToken
		}
	}

	if execution.VariablesFile != "" {
		output.PrintLogf("%s Creating variables file...", ui.IconWorld)
		file := filepath.Join(r.dir, "params-file")
		if err = os.WriteFile(file, []byte(execution.VariablesFile), 0666); err != nil {
			output.PrintLogf("%s Could not create variables file %s: %s", ui.IconCross, file, err.Error())
			return result, errors.Errorf("could not create variables file %s: %v", file, err)
		}
		output.PrintLogf("%s Variables file created", ui.IconCheckMark)
	}

	_, err = r.Fetcher.Fetch(execution.Content)
	if err != nil {
		output.PrintLogf("%s Could not fetch test content: %s", ui.IconCross, err.Error())
		return result, errors.Errorf("could not fetch test content: %v", err)
	}

	// add copy files in case object storage is set
	if params.Endpoint != "" {
		output.PrintLogf("%s Fetching uploads from object store %s...", ui.IconFile, params.Endpoint)
		minioClient := minio.NewClient(params.Endpoint, params.AccessKeyID, params.SecretAccessKey, params.Region, params.Token, params.Bucket, params.Ssl)
		fp := content.NewCopyFilesPlacer(minioClient)
		fp.PlaceFiles(execution.TestName, execution.BucketName)
	}

	output.PrintLogf("%s Setting up access to files in %s", ui.IconFile, r.dir)
	_, err = executor.Run(r.dir, "chmod", nil, []string{"-R", "777", "."}...)
	if err != nil {
		output.PrintLogf("%s Could not chmod for data dir: %s", ui.IconCross, err.Error())
	}

	if execution.ArtifactRequest != nil {
		_, err = executor.Run(execution.ArtifactRequest.VolumeMountPath, "chmod", nil, []string{"-R", "777", "."}...)
		if err != nil {
			output.PrintLogf("%s Could not chmod for artifacts dir: %s", ui.IconCross, err.Error())
		}
	}
	output.PrintLogf("%s Access to files enabled", ui.IconCheckMark)

	output.PrintLogf("%s Initialization successful", ui.IconCheckMark)
	return testkube.NewPendingExecutionResult(), nil
}

// GetType returns runner type
func (r *InitRunner) GetType() runner.Type {
	return runner.TypeInit
}
