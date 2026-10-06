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

type DockerInfo struct {
	NCPU          int      `json:"NCPU"`
	MemTotal      int64    `json:"MemTotal"`
	DockerRootDir string   `json:"DockerRootDir"`
	ServerErrors  []string `json:"ServerErrors"`
}

type Docker interface {
	Info(ctx context.Context) (DockerInfo, error)
}

type dockerCLI struct{}

func (dockerCLI) Info(ctx context.Context) (DockerInfo, error) {
	// docker info hangs while the daemon is still starting.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .}}")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if reason := strings.TrimSpace(stderr.String()); reason != "" {
			return DockerInfo{}, errors.New(reason)
		}
		return DockerInfo{}, err
	}
	var info DockerInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return DockerInfo{}, err
	}
	if len(info.ServerErrors) > 0 {
		return DockerInfo{}, errors.New(strings.Join(info.ServerErrors, "; "))
	}
	return info, nil
}
