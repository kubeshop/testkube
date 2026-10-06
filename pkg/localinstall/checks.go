package localinstall

import (
	"context"
	"os/exec"
	"time"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Result struct {
	Name   string
	Status Status
	Detail string
	Fix    string
}

// Swapped in tests.
var (
	lookPath      = exec.LookPath
	dockerRunning = func(ctx context.Context) bool {
		// docker info hangs while the daemon is still starting.
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, "docker", "info").Run() == nil
	}
)

// We install these ourselves, so missing is only a warning.
var installableTools = []string{"kubectl", "helm", "kind"}

func CheckTools(ctx context.Context) []Result {
	results := []Result{checkDocker(ctx)}
	for _, name := range installableTools {
		results = append(results, checkInstallable(name))
	}
	return results
}

func HasFailure(results []Result) bool {
	for _, r := range results {
		if r.Status == StatusFail {
			return true
		}
	}
	return false
}

func checkDocker(ctx context.Context) Result {
	path, err := lookPath("docker")
	if err != nil {
		return Result{Name: "docker", Status: StatusFail, Detail: "not found", Fix: "Install Docker: https://docs.docker.com/get-docker/"}
	}
	if !dockerRunning(ctx) {
		return Result{Name: "docker", Status: StatusFail, Detail: "not running", Fix: "Start Docker, then run the installer again"}
	}
	return Result{Name: "docker", Status: StatusPass, Detail: path}
}

func checkInstallable(name string) Result {
	path, err := lookPath(name)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: "not found, will be installed"}
	}
	return Result{Name: name, Status: StatusPass, Detail: path}
}
