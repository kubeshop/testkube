package runner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewRunner creates init runner
func NewRunner(params envs.Params) *InitRunner {
	dir := os.Getenv("RUNNER_DATADIR")
	return &InitRunner{
		Fetcher: content.NewFetcher(dir),
		Params:  params,
		dir:     dir,
	}
}

// InitRunner prepares data for executor
type InitRunner struct {
	Fetcher content.ContentFetcher
	Params  envs.Params
	dir     string
}

var _ runner.Runner = &InitRunner{}

// Run prepares data for executor
func (r *InitRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLogf("%s Initializing...", ui.IconTruck)

	gitUsername := r.Params.GitUsername
	gitToken := r.Params.GitToken

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

	if execution.PreRunScript != "" || execution.PostRunScript != "" || execution.ContainerEntrypoint != "" {
		output.PrintLogf("%s Creating entrypoint script...", ui.IconWorld)
		file := filepath.Join(r.dir, "entrypoint.sh")
		scripts := []string{execution.PreRunScript, execution.ContainerEntrypoint, execution.PostRunScript}
		var data string
		for _, script := range scripts {
			data += script
			if script != "" {
				data += "\n"
			}
		}

		if err = os.WriteFile(file, []byte(data), 0755); err != nil {
			output.PrintLogf("%s Could not create entrypoint script %s: %s", ui.IconCross, file, err.Error())
			return result, errors.Errorf("could not create entrypoint script %s: %v", file, err)
		}
		output.PrintLogf("%s Entrypoint script created", ui.IconCheckMark)
	}

	// TODO: write a proper cloud implementation
	// add copy files in case object storage is set
	if r.Params.Endpoint != "" && !r.Params.CloudMode {
		output.PrintLogf("%s Fetching uploads from object store %s...", ui.IconFile, r.Params.Endpoint)
		minioClient := minio.NewClient(r.Params.Endpoint, r.Params.AccessKeyID, r.Params.SecretAccessKey, r.Params.Region, r.Params.Token, r.Params.Bucket, r.Params.Ssl)
		fp := content.NewCopyFilesPlacer(minioClient)
		fp.PlaceFiles(ctx, execution.TestName, execution.BucketName)
	} else if r.Params.CloudMode {
		output.PrintLogf("%s Copy files functionality is currently not supported in cloud mode", ui.IconWarning)
	}

	output.PrintLogf("%s Setting up access to files in %s", ui.IconFile, r.dir)
	_, err = executor.Run(r.dir, "chmod", nil, []string{"-R", "777", "."}...)
	if err != nil {
		output.PrintLogf("%s Could not chmod for data dir: %s", ui.IconCross, err.Error())
	}

	if execution.ArtifactRequest != nil {
		mountPath := filepath.Join(r.Params.DataDir, "artifacts")
		if execution.ArtifactRequest.VolumeMountPath != "" {
			mountPath = execution.ArtifactRequest.VolumeMountPath
		}

		_, err = executor.Run(mountPath, "chmod", nil, []string{"-R", "777", "."}...)
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
