package export

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
)

var (
	outputPathJSONPattern = regexp.MustCompile(`"outputPath"\s*:\s*"([^"]+)"`)
	outputPathLogPattern  = regexp.MustCompile(`outputPath[=:\s]+(/[^\s",]+)`)
	downloadLinePattern   = regexp.MustCompile(`Download:\s+kubectl\s+-n\s+\S+\s+cp\s+\S+:(\S+)`)
)

func ParseOutputPath(logLine string) (string, bool) {
	if matches := outputPathJSONPattern.FindStringSubmatch(logLine); len(matches) == 2 {
		return matches[1], true
	}
	if matches := outputPathLogPattern.FindStringSubmatch(logLine); len(matches) == 2 {
		return matches[1], true
	}
	if matches := downloadLinePattern.FindStringSubmatch(logLine); len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

func WaitAndDownload(ctx context.Context, opts Options, jobName string) (string, string, *common.CLIError) {
	kubectlPath, cliErr := common.LookupKubectlPath()
	if cliErr != nil {
		return "", "", cliErr
	}

	if opts.DryRun {
		return opts.Output, "dry-run-pod", nil
	}

	timeout, err := time.ParseDuration(opts.Timeout)
	if err != nil {
		return "", "", common.NewCLIError(
			common.TKErrInvalidRuntimeParameter,
			"Invalid timeout",
			"Provide a valid duration such as 15m or 30m",
			err,
		)
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	podName, remotePath, cliErr := waitForExportComplete(waitCtx, kubectlPath, opts, jobName)
	if cliErr != nil {
		return "", "", cliErr
	}

	localPath := opts.Output
	if localPath == "" {
		localPath = defaultLocalOutput(remotePath)
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return "", "", common.NewCLIError(
			common.TKErrKubectlCommandFailed,
			"Failed to create output directory",
			"Choose a writable --output path",
			err,
		)
	}

	cpArgs := kubectlBaseArgs(opts, "cp", fmt.Sprintf("%s:%s", podName, remotePath), localPath)
	if _, cliErr := common.RunKubectlCommand(kubectlPath, cpArgs); cliErr != nil {
		return "", "", cliErr
	}

	return localPath, podName, nil
}

func waitForExportComplete(ctx context.Context, kubectlPath string, opts Options, jobName string) (podName, remotePath string, cliErr *common.CLIError) {
	podName, cliErr = waitForJobPod(ctx, kubectlPath, opts, jobName)
	if cliErr != nil {
		return "", "", cliErr
	}

	logArgs := kubectlBaseArgs(opts, "logs", "-f", podName)
	cmd := exec.CommandContext(ctx, kubectlPath, logArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", common.NewCLIError(common.TKErrKubectlCommandFailed, "Failed to stream pod logs", "Retry the command or inspect the Job manually with kubectl logs", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", "", common.NewCLIError(common.TKErrKubectlCommandFailed, "Failed to stream pod logs", "Retry the command or inspect the Job manually with kubectl logs", err)
	}

	if err := cmd.Start(); err != nil {
		return "", "", common.NewCLIError(common.TKErrKubectlCommandFailed, "Failed to stream pod logs", "Retry the command or inspect the Job manually with kubectl logs", err)
	}

	done := make(chan string, 1)
	go func() {
		defer close(done)
		var exportPath string
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			line := scanner.Text()
			if path, ok := ParseOutputPath(line); ok {
				exportPath = path
			}
			if strings.Contains(line, "usage export complete") && exportPath != "" {
				done <- exportPath
				return
			}
		}
		done <- exportPath
	}()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return "", "", common.NewCLIError(
			common.TKErrKubectlCommandFailed,
			"Timed out waiting for usage export",
			fmt.Sprintf("Increase --timeout (current %s) or inspect logs: kubectl logs %s", opts.Timeout, podName),
			ctx.Err(),
		)
	case exportPath := <-done:
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if exportPath == "" {
			exportPath, cliErr = discoverOutputPath(kubectlPath, opts, podName)
			if cliErr != nil {
				return "", "", cliErr
			}
			return podName, exportPath, nil
		}
		return podName, exportPath, nil
	}
}

func waitForJobPod(ctx context.Context, kubectlPath string, opts Options, jobName string) (string, *common.CLIError) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		args := kubectlBaseArgs(opts, "get", "pods",
			"-l", fmt.Sprintf("job-name=%s", jobName),
			"-o", "jsonpath={.items[0].metadata.name}",
		)
		podName, cliErr := common.RunKubectlCommand(kubectlPath, args)
		if cliErr == nil && strings.TrimSpace(podName) != "" {
			return strings.TrimSpace(podName), nil
		}

		select {
		case <-ctx.Done():
			return "", common.NewCLIError(
				common.TKErrKubectlCommandFailed,
				"Timed out waiting for usage export pod",
				fmt.Sprintf("Inspect the Job: kubectl describe job %s", jobName),
				ctx.Err(),
			)
		case <-ticker.C:
		}
	}
}

func discoverOutputPath(kubectlPath string, opts Options, podName string) (string, *common.CLIError) {
	args := kubectlBaseArgs(opts, "exec", podName, "--", "sh", "-c", "ls -1 /output/*.zip 2>/dev/null | head -1")
	output, cliErr := common.RunKubectlCommand(kubectlPath, args)
	if cliErr != nil {
		return "", cliErr
	}
	path := strings.TrimSpace(output)
	if path == "" {
		return "", common.NewCLIError(
			common.TKErrKubectlCommandFailed,
			"Could not locate usage export zip in pod",
			"Inspect pod logs and copy manually from /output",
			fmt.Errorf("no zip found in /output"),
		)
	}
	return path, nil
}

func defaultLocalOutput(remotePath string) string {
	base := filepath.Base(remotePath)
	if base == "" || base == "." {
		base = fmt.Sprintf("plan-usage-%s.zip", time.Now().UTC().Format("20060102T150405Z"))
	}
	return base
}
