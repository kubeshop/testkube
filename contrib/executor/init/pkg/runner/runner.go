package runner

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/containerexecutor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	defaultShell      = "/bin/sh"
	preRunScriptName  = "prerun.sh"
	commandScriptName = "command.sh"
	postRunScriptName = "postrun.sh"
)

// NewRunner creates init runner
func NewRunner(params envs.Params) *InitRunner {
	return &InitRunner{
		Fetcher: content.NewFetcher(params.DataDir),
		Params:  params,
	}
}

// InitRunner prepares data for executor
type InitRunner struct {
	Fetcher content.ContentFetcher
	Params  envs.Params
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
		file := filepath.Join(r.Params.DataDir, "params-file")
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
		shell := defaultShell
		if execution.ContainerShell != "" {
			shell = execution.ContainerShell
		}

		shebang := "#!" + shell + "\nset -e\n"
		// No set -e so that we can run the post-run script even if the command fails
		entrypoint := "#!" + shell + "\n"
		command := shebang
		preRunScript := shebang
		postRunScript := shebang

		if execution.PreRunScript != "" {
			if execution.SourceScripts {
				entrypoint += ". "
			}

			entrypoint += strconv.Quote(filepath.Join(r.Params.DataDir, preRunScriptName)) + "\n"
			entrypoint += "prerun_exit_code=$?\nif [ $prerun_exit_code -ne 0 ]; then\n  exit $prerun_exit_code\nfi\n"
			preRunScript += execution.PreRunScript
		}

		if len(execution.Command) != 0 {
			if execution.SourceScripts {
				entrypoint += ". "
			}

			entrypoint += strconv.Quote(filepath.Join(r.Params.DataDir, commandScriptName)) + " $@\n"
			entrypoint += "command_exit_code=$?\n"
			command += strings.Join(execution.Command, " ")
			command += " \"$@\"\n"
		}

		if execution.PostRunScript != "" {
			if execution.SourceScripts {
				entrypoint += ". "
			}

			entrypoint += strconv.Quote(filepath.Join(r.Params.DataDir, postRunScriptName)) + "\n"
			entrypoint += "postrun_exit_code=$?\n"
			postRunScript += execution.PostRunScript
		}

		if len(execution.Command) != 0 {
			entrypoint += "if [ $command_exit_code -ne 0 ]; then\n  exit $command_exit_code\nfi\n"
		}

		if execution.PostRunScript != "" {
			entrypoint += "exit $postrun_exit_code\n"
		}
		var scripts = []struct {
			dir     string
			file    string
			data    string
			comment string
		}{
			{r.Params.DataDir, preRunScriptName, preRunScript, "prerun"},
			{r.Params.DataDir, commandScriptName, command, "command"},
			{r.Params.DataDir, postRunScriptName, postRunScript, "postrun"},
			{r.Params.DataDir, containerexecutor.EntrypointScriptName, entrypoint, "entrypoint"},
		}

		for _, script := range scripts {
			if script.data == "" {
				continue
			}

			file := filepath.Join(script.dir, script.file)
			output.PrintLogf("%s Creating %s script...", ui.IconWorld, script.comment)
			if err = os.WriteFile(file, []byte(script.data), 0755); err != nil {
				output.PrintLogf("%s Could not create %s script %s: %s", ui.IconCross, script.comment, file, err.Error())
				return result, errors.Errorf("could not create %s script %s: %v", script.comment, file, err)
			}
			output.PrintLogf("%s %s script created", ui.IconCheckMark, script.comment)
		}
	}

	// TODO: write a proper cloud implementation
	if r.Params.Endpoint != "" && !r.Params.ProMode {
		output.PrintLogf("%s Fetching uploads from object store %s...", ui.IconFile, r.Params.Endpoint)
		opts := minio.GetTLSOptions(r.Params.Ssl, r.Params.SkipVerify, r.Params.CertFile, r.Params.KeyFile, r.Params.CAFile)
		minioClient := minio.NewClient(r.Params.Endpoint, r.Params.AccessKeyID, r.Params.SecretAccessKey, r.Params.Region, r.Params.Token, r.Params.Bucket, opts...)
		fp := content.NewCopyFilesPlacer(minioClient)
		fp.PlaceFiles(ctx, execution.TestName, execution.BucketName)
	} else if r.Params.ProMode {
		output.PrintLogf("%s Copy files functionality is currently not supported in cloud mode", ui.IconWarning)
	}

	output.PrintLogf("%s Setting up access to files in %s", ui.IconFile, r.Params.DataDir)
	_, err = executor.Run(r.Params.DataDir, "chmod", nil, []string{"-R", "777", "."}...)
	if err != nil {
		output.PrintLogf("%s Could not chmod for data dir: %s", ui.IconCross, err.Error())
	}

	if execution.ArtifactRequest != nil && execution.ArtifactRequest.StorageClassName != "" {
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

	if len(execution.DownloadArtifactExecutionIDs) != 0 || len(execution.DownloadArtifactTestNames) != 0 {
		downloadedArtifacts := filepath.Join(r.Params.DataDir, "downloaded-artifacts")
		options := client.Options{
			ApiUri: r.Params.APIURI,
		}

		c, err := client.GetClient(client.ClientDirect, options)
		if err != nil {
			output.PrintLogf("%s Could not get client: %s", ui.IconCross, err.Error())
		} else {
			for _, id := range execution.DownloadArtifactExecutionIDs {
				execution, err := c.GetExecution(id)
				if err != nil {
					output.PrintLogf("%s Could not get execution: %s", ui.IconCross, err.Error())
					continue
				}

				if err = downloadArtifacts(id, filepath.Join(downloadedArtifacts, execution.TestName+"-"+id), c); err != nil {
					output.PrintLogf("%s Could not download execution artifact: %s", ui.IconCross, err.Error())
				}
			}

			for _, name := range execution.DownloadArtifactTestNames {
				test, err := c.GetTestWithExecution(name)
				if err != nil {
					output.PrintLogf("%s Could not get test with execution: %s", ui.IconCross, err.Error())
					continue
				}

				if test.LatestExecution != nil {
					id := test.LatestExecution.Id
					if err = downloadArtifacts(id, filepath.Join(downloadedArtifacts, name+"-"+id), c); err != nil {
						output.PrintLogf("%s Could not download test artifact: %s", ui.IconCross, err.Error())
					}
				}
			}
		}
	}

	output.PrintLogf("%s Initialization successful", ui.IconCheckMark)
	return testkube.NewPendingExecutionResult(), nil
}

func downloadArtifacts(id, dir string, c client.Client) error {
	artifacts, err := c.GetExecutionArtifacts(id)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	if len(artifacts) > 0 {
		output.PrintLogf("%s Getting %d artifacts...", ui.IconWorld, len(artifacts))
		for _, artifact := range artifacts {
			f, err := c.DownloadFile(id, artifact.Name, dir)
			if err != nil {
				return err
			}

			output.PrintLogf("%s Downloading file %s...", ui.IconWorld, f)
		}
	}

	return nil
}

// GetType returns runner type
func (r *InitRunner) GetType() runner.Type {
	return runner.TypeInit
}
