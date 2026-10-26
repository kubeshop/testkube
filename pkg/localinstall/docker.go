package localinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"
)

type dockerInfo struct {
	NCPU          int      `json:"NCPU"`
	MemTotal      int64    `json:"MemTotal"`
	DockerRootDir string   `json:"DockerRootDir"`
	ServerErrors  []string `json:"ServerErrors"`
	ServerVersion string   `json:"ServerVersion"`
	// Names the engine: Docker Desktop, OrbStack, Colima.
	OperatingSystem string `json:"OperatingSystem"`
	KernelVersion   string `json:"KernelVersion"`
	// Colima's VM is named after it.
	Name string `json:"Name"`
	// Only Podman's docker stand-in answers with this.
	PodmanHost json.RawMessage `json:"host"`
}

type dockerClient interface {
	Info(ctx context.Context) (dockerInfo, error)
}

var errDockerTimeout = errors.New("docker did not answer within 10 seconds")

type dockerCLI struct{}

func (dockerCLI) Info(ctx context.Context) (dockerInfo, error) {
	// docker info hangs while the daemon is still starting.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .}}")
	// Children holding stdout open would outlive the timeout.
	cmd.WaitDelay = time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return dockerInfo{}, errDockerTimeout
		}
		if reason := strings.TrimSpace(stderr.String()); reason != "" {
			return dockerInfo{}, errors.New(reason)
		}
		return dockerInfo{}, err
	}
	var info dockerInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return dockerInfo{}, err
	}
	if len(info.ServerErrors) > 0 {
		return dockerInfo{}, errors.New(strings.Join(info.ServerErrors, "; "))
	}
	return info, nil
}

// Zero when Docker couldn't be read; fixes then stay generic.
type DockerResources struct {
	Engine  string
	Version string
	CPUs    int
	Memory  uint64
}

func (d DockerResources) DockerDesktop() bool { return d.Engine == "Docker Desktop" }
func (d DockerResources) Colima() bool        { return strings.HasPrefix(d.Engine, "colima") }
func (d DockerResources) MemoryText() string  { return formatGB(d.Memory) }

// Same bar as the checks: Docker reports ~3% low.
func (d DockerResources) Enough(cpu bool) bool {
	if cpu {
		return d.CPUs >= minCPUs
	}
	return d.Memory >= minMemoryReported
}

const (
	NeededCPUs     = minCPUs
	NeededMemoryGB = minMemory / gigabyte
)

// The failure path has no Checker; one more call is cheap.
func ReadDockerResources(ctx context.Context) DockerResources {
	info, err := dockerCLI{}.Info(ctx)
	if err != nil || info.NCPU == 0 || info.MemTotal == 0 {
		return DockerResources{}
	}
	engine := info.OperatingSystem
	if strings.HasPrefix(info.Name, "colima") {
		engine = info.Name
	}
	return DockerResources{Engine: engine, Version: info.ServerVersion, CPUs: info.NCPU, Memory: uint64(info.MemTotal)}
}
