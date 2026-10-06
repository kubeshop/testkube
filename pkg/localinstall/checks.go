package localinstall

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

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

const (
	gigabyte      = 1 << 30
	minCPUs       = 4
	minMemory     = 6 * gigabyte
	minFreeDisk   = 10 * gigabyte
	resourcesHint = "Testkube may run slowly or fail to start"

	// Docker reports ~3% under its setting: 6 GB shows ~5.8.
	minMemoryReported = 55 * gigabyte / 10

	systemDockerSocket  = "/var/run/docker.sock"
	dockerPermissionFix = "Your user can't use Docker yet. Run:\n" +
		"  sudo usermod -aG docker $USER\n" +
		"  newgrp docker\n" +
		"Then run the installer again. Details: https://docs.docker.com/engine/install/linux-postinstall/"
)

// We install these ourselves, so missing is only a warning.
var installableTools = []string{"kubectl", "helm", "kind"}

type host interface {
	LookPath(name string) (string, error)
	FreeDiskAt(path string) (free uint64, visible bool, err error)
}

type realHost struct{}

func (realHost) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (realHost) FreeDiskAt(path string) (uint64, bool, error) {
	return freeDiskAt(path)
}

type Checker struct {
	host   host
	docker dockerClient
	info   *dockerInfo
}

func NewChecker() *Checker {
	return &Checker{host: realHost{}, docker: dockerCLI{}}
}

func (c *Checker) CheckTools(ctx context.Context) []Result {
	results := []Result{c.checkDocker(ctx)}
	for _, name := range installableTools {
		results = append(results, c.checkInstallable(name))
	}
	return results
}

// Machine size only warns: users may continue on smaller machines.
func (c *Checker) CheckMachine(ctx context.Context) []Result {
	info, err := c.readDockerInfo(ctx)
	// Docker-compatible engines like Podman report other fields, so zeros.
	if err != nil || info.NCPU == 0 || info.MemTotal == 0 {
		return []Result{
			{Name: "cpu", Status: StatusWarn, Detail: "could not read from Docker"},
			{Name: "memory", Status: StatusWarn, Detail: "could not read from Docker"},
			{Name: "disk", Status: StatusWarn, Detail: "could not read free space"},
		}
	}
	// Docker Desktop only gets a share of the host.
	results := []Result{
		minimumResult("cpu", info.NCPU >= minCPUs, fmt.Sprintf("%d cores", info.NCPU), fmt.Sprintf("needs %d", minCPUs)),
		minimumResult("memory", info.MemTotal >= minMemoryReported, formatGB(uint64(info.MemTotal)), "needs "+formatGB(minMemory)),
	}

	if info.DockerRootDir == "" {
		return append(results, Result{Name: "disk", Status: StatusWarn, Detail: "could not read free space"})
	}
	free, visible, err := c.host.FreeDiskAt(info.DockerRootDir)
	switch {
	case err != nil:
		return append(results, Result{Name: "disk", Status: StatusWarn, Detail: "could not read free space"})
	case !visible:
		return results
	}
	return append(results, minimumResult("disk", free >= minFreeDisk, formatGB(free)+" free", "needs "+formatGB(minFreeDisk)))
}

func HasFailure(results []Result) bool {
	for _, r := range results {
		if r.Status == StatusFail {
			return true
		}
	}
	return false
}

// Both check groups read one docker info call.
func (c *Checker) readDockerInfo(ctx context.Context) (dockerInfo, error) {
	if c.info != nil {
		return *c.info, nil
	}
	info, err := c.docker.Info(ctx)
	if err != nil {
		return dockerInfo{}, err
	}
	c.info = &info
	return info, nil
}

func (c *Checker) checkDocker(ctx context.Context) Result {
	path, err := c.host.LookPath("docker")
	if err != nil {
		return Result{Name: "docker", Status: StatusFail, Detail: "not found", Fix: "Install Docker: https://docs.docker.com/get-docker/"}
	}
	if _, err := c.readDockerInfo(ctx); err != nil {
		if errors.Is(err, errDockerTimeout) {
			return Result{Name: "docker", Status: StatusFail, Detail: "not answering", Fix: "Docker did not answer within 10 seconds. It may still be starting; wait a moment, then run again"}
		}
		// The docker group only grants access to the system socket.
		if strings.Contains(err.Error(), "permission denied") && strings.Contains(err.Error(), systemDockerSocket) {
			return Result{Name: "docker", Status: StatusFail, Detail: "permission denied", Fix: dockerPermissionFix}
		}
		return Result{Name: "docker", Status: StatusFail, Detail: "not reachable", Fix: err.Error() + "\nStart Docker, then run the installer again"}
	}
	return Result{Name: "docker", Status: StatusPass, Detail: path}
}

func (c *Checker) checkInstallable(name string) Result {
	path, err := c.host.LookPath(name)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: "not found, will be installed"}
	}
	return Result{Name: name, Status: StatusPass, Detail: path}
}

// Docker Desktop's data dir lives inside its VM, invisible here.
func freeDiskAt(root string) (free uint64, visible bool, err error) {
	if _, err := os.Stat(root); err != nil {
		// Only absence means VM-hidden; other errors must still warn.
		if errors.Is(err, fs.ErrNotExist) {
			return 0, false, nil
		}
		return 0, true, err
	}
	usage, err := disk.Usage(root)
	if err != nil {
		return 0, true, err
	}
	return usage.Free, true, nil
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
