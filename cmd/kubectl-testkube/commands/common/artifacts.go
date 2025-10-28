package common

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	artifactsFormatFolder  = "folder"
	artifactsFormatArchive = "archive"
	maxArgSize             = int64(131072) // maximum argument size in linux-based systems is 128 KiB
)

func DownloadTestWorkflowArtifacts(id, dir, format string, masks []string, client client.Client, outputPretty bool) {
	artifacts, err := client.GetTestWorkflowExecutionArtifacts(id)
	ui.ExitOnError("getting artifacts", err)

	downloadFile := func(artifact testkube.Artifact, dir string) (string, error) {
		return client.DownloadTestWorkflowArtifact(id, artifact.Name, dir)
	}
	downloadArchive := func(dir string, masks []string) (string, error) {
		return client.DownloadTestWorkflowArtifactArchive(id, dir, masks)
	}
	downloadArtifacts(dir, format, masks, artifacts, downloadFile, downloadArchive, outputPretty)
}

func downloadArtifacts(
	dir, format string,
	masks []string,
	artifacts testkube.Artifacts,
	downloadFile func(artifact testkube.Artifact, dir string) (string, error),
	downloadArchive func(dir string, masks []string) (string, error),
	outputPretty bool,
) {
	err := os.MkdirAll(dir, os.ModePerm)
	ui.ExitOnError("creating dir "+dir, err)

	if len(artifacts) > 0 && outputPretty {
		ui.Info("Getting artifacts", fmt.Sprintf("count = %d", len(artifacts)), "\n")
	}

	if format != artifactsFormatFolder && format != artifactsFormatArchive {
		ui.Failf("invalid artifacts format: %s. use one of folder|archive", format)
	}

	var regexps []*regexp.Regexp
	for _, mask := range masks {
		values := strings.Split(mask, ",")
		for _, value := range values {
			re, err := regexp.Compile(value)
			ui.ExitOnError("checking mask "+value, err)

			regexps = append(regexps, re)
		}
	}

	if format == artifactsFormatFolder {
		for _, artifact := range artifacts {
			found := len(regexps) == 0
			for i := range regexps {
				if found = regexps[i].MatchString(artifact.Name); found {
					break
				}
			}

			if !found {
				continue
			}

			f, err := downloadFile(artifact, dir)
			ui.ExitOnError("downloading file: "+f, err)
			if outputPretty {
				ui.Warn(" - downloading file ", f)
			}
		}
	}

	if format == artifactsFormatArchive {
		const readinessCheckTimeout = time.Second
		ticker := time.NewTicker(readinessCheckTimeout)
		defer ticker.Stop()

		ch := make(chan string)
		defer close(ch)

		go func() {
			f, err := downloadArchive(dir, masks)
			ui.ExitOnError("downloading archive: "+f, err)

			ch <- f
		}()

		var archive string
		if outputPretty {
			ui.Warn(" - preparing archive ")
		}

	outloop:
		for {
			select {
			case <-ticker.C:
				if outputPretty {
					ui.PrintDot()
				}
			case archive = <-ch:
				if outputPretty {
					ui.NL()
				}
				break outloop
			}
		}

		if outputPretty {
			ui.Warn(" - downloading archive ", archive)
		}
	}

	if outputPretty {
		ui.NL()
		ui.NL()
	}
}

// readCopyFiles reads files
func readCopyFiles(copyFiles []string) (map[string]string, error) {
	files := map[string]string{}
	for _, f := range copyFiles {
		paths := strings.Split(f, ":")
		if len(paths) != 2 {
			return nil, fmt.Errorf("invalid file format, expecting sourcePath:destinationPath")
		}
		contents, err := os.ReadFile(paths[0])
		if err != nil {
			return nil, fmt.Errorf("could not read executor copy file: %w", err)
		}
		files[paths[1]] = string(contents)
	}
	return files, nil
}

// mergeCopyFiles merges the lists of files to be copied into the running test
// the files set on execution overwrite the files set on test levels
func mergeCopyFiles(testFiles []string, executionFiles []string) ([]string, error) {
	if len(testFiles) == 0 {
		return executionFiles, nil
	}

	if len(executionFiles) == 0 {
		return testFiles, nil
	}

	files := map[string]string{}

	for _, fileMapping := range testFiles {
		fPair := strings.Split(fileMapping, ":")
		if len(fPair) != 2 {
			return []string{}, fmt.Errorf("invalid copy file mapping, expected source:destination, got: %s", fileMapping)
		}
		files[fPair[1]] = fPair[0]
	}

	for _, fileMapping := range executionFiles {
		fPair := strings.Split(fileMapping, ":")
		if len(fPair) != 2 {
			return []string{}, fmt.Errorf("invalid copy file mapping, expected source:destination, got: %s", fileMapping)
		}
		files[fPair[1]] = fPair[0]
	}

	result := []string{}
	for destination, source := range files {
		result = append(result, fmt.Sprintf("%s:%s", source, destination))
	}

	return result, nil
}

// isFileTooBigForCLI checks the file size found on path and compares it with maxArgSize
func isFileTooBigForCLI(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("could not open file %s: %w", path, err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s could not close file %s: %v", ui.IconWarning, f.Name(), err))
		}
	}()

	fileInfo, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("could not get info on file %s: %w", path, err)
	}

	return fileInfo.Size() < maxArgSize, nil
}

// PrepareVariablesFile reads variables file, or if the file size is too big
// it uploads them
func PrepareVariablesFile(client client.Client, parentName string, parentType client.TestingType, filePath string, timeout time.Duration) (string, bool, error) {
	isFileSmall, err := isFileTooBigForCLI(filePath)
	if err != nil {
		return "", false, fmt.Errorf("could not determine if variables file %s needs to be uploaded: %w", filePath, err)
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", false, fmt.Errorf("could not read file %s: %w", filePath, err)
	}
	if isFileSmall {
		return string(b), false, nil
	}

	fileName := path.Base(filePath)

	err = client.UploadFile(parentName, parentType, fileName, b, timeout)
	if err != nil {
		return "", false, fmt.Errorf("could not upload variables file for %v with name %s: %w", parentType, parentName, err)
	}
	return fileName, true, nil
}
