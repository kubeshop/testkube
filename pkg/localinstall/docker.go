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
