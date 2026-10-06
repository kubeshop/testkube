package localinstall

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
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
	// Docker Desktop only gets a share of the host.
	dockerResources = func(ctx context.Context) (cpus int, memoryBytes int64, err error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.NCPU}} {{.MemTotal}}").Output()
		if err != nil {
			return 0, 0, err
		}
		fields := strings.Fields(string(out))
		if len(fields) != 2 {
			return 0, 0, fmt.Errorf("unexpected docker info output: %q", out)
		}
		if cpus, err = strconv.Atoi(fields[0]); err != nil {
			return 0, 0, err
		}
		memoryBytes, err = strconv.ParseInt(fields[1], 10, 64)
		return cpus, memoryBytes, err
	}
	freeDiskBytes = func() (uint64, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0, err
		}
		usage, err := disk.Usage(home)
		if err != nil {
			return 0, err
		}
		return usage.Free, nil
	}
)

const (
	gigabyte      = 1 << 30
	minCPUs       = 4
	minMemory     = 6 * gigabyte
	minFreeDisk   = 10 * gigabyte
	resourcesHint = "Testkube may run slowly or fail to start"
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

// Machine size only warns: users may continue on smaller machines.
func CheckMachine(ctx context.Context) []Result {
	results := make([]Result, 0, 3)
	cpus, memoryBytes, err := dockerResources(ctx)
	if err != nil {
		results = append(results,
			Result{Name: "cpu", Status: StatusWarn, Detail: "could not read from Docker"},
			Result{Name: "memory", Status: StatusWarn, Detail: "could not read from Docker"})
	} else {
		results = append(results,
			minimumResult("cpu", cpus >= minCPUs, fmt.Sprintf("%d cores", cpus), fmt.Sprintf("needs %d", minCPUs)),
			minimumResult("memory", memoryBytes >= minMemory, formatGB(uint64(memoryBytes)), "needs "+formatGB(minMemory)))
	}

	free, err := freeDiskBytes()
	if err != nil {
		return append(results, Result{Name: "disk", Status: StatusWarn, Detail: "could not read free space"})
	}
	return append(results, minimumResult("disk", free >= minFreeDisk, formatGB(free)+" free", "needs "+formatGB(minFreeDisk)))
}

func minimumResult(name string, ok bool, have, need string) Result {
	if ok {
		return Result{Name: name, Status: StatusPass, Detail: have}
	}
	return Result{Name: name, Status: StatusWarn, Detail: have + ", " + need, Fix: resourcesHint}
}

func formatGB(bytes uint64) string {
	return fmt.Sprintf("%.1f GB", float64(bytes)/gigabyte)
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
