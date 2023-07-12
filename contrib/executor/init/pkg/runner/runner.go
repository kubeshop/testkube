package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"

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

const defaultShell = "/bin/sh"

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

	if execution.PreRunScript != "" || execution.PostRunScript != "" {
		command := "#!" + defaultShell
		if execution.ContainerShell != "" {
			command = "#!" + execution.ContainerShell
		}
		command += "\n"

		if execution.PreRunScript != "" {
			command += filepath.Join(r.dir, "prerun.sh") + "\n"
		}

		if len(execution.Command) != 0 {
			command += strings.Join(execution.Command, " ")
			command += " \"$@\"\n"
		}

		if execution.PostRunScript != "" {
			command += filepath.Join(r.dir, "postrun.sh") + "\n"
		}

		var scripts = []struct {
			file    string
			data    string
			comment string
		}{
			{"prerun.sh", execution.PreRunScript, "prerun"},
			{"entrypoint.sh", command, "entrypoint"},
			{"postrun.sh", execution.PostRunScript, "postrun"},
		}

		for _, script := range scripts {
			if script.data == "" {
				continue
			}

			file := filepath.Join(r.dir, script.file)
			output.PrintLogf("%s Creating %s script...", ui.IconWorld, script.comment)
			if err = os.WriteFile(file, []byte(script.data), 0755); err != nil {
				output.PrintLogf("%s Could not create %s script %s: %s", ui.IconCross, script.comment, file, err.Error())
				return result, errors.Errorf("could not create %s script %s: %v", script.comment, file, err)
			}
			output.PrintLogf("%s %s script created", ui.IconCheckMark, script.comment)
		}
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
