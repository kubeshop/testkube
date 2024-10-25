// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kubeshop/testkube/pkg/tmp"
)

type Binary struct {
	mainPath         string
	outputPath       string
	operatingSystem  string
	procArchitecture string

	hash    string
	buildMu sync.RWMutex
}

func NewBinary(mainPath, operatingSystem, procArchitecture string) *Binary {
	return &Binary{
		mainPath:         mainPath,
		outputPath:       tmp.Name(),
		operatingSystem:  operatingSystem,
		procArchitecture: procArchitecture,
	}
}

func (b *Binary) updateHash() error {
	f, err := os.Open(b.outputPath)
	if err != nil {
		return fmt.Errorf("failed to get hash: reading binary: %s", err.Error())
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to get hash: %s", err.Error())
	}

	b.hash = fmt.Sprintf("%x", h.Sum(nil))
	return nil
}

func (b *Binary) Hash() string {
	b.buildMu.RLock()
	defer b.buildMu.RUnlock()
	return b.hash
}

func (b *Binary) Path() string {
	b.buildMu.RLock()
	defer b.buildMu.RUnlock()
	return b.outputPath
}

func (b *Binary) Build(ctx context.Context) (string, error) {
	b.buildMu.Lock()
	defer b.buildMu.Unlock()

	cmd := exec.Command(
		"go", "build",
		"-o", b.outputPath,
		fmt.Sprintf("-ldflags=%s", strings.Join([]string{
			"-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=",
			"-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=",
			"-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=",
			"-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=",
			"-X github.com/kubeshop/testkube/internal/pkg/api.Version=devbox",
			"-X github.com/kubeshop/testkube/internal/pkg/api.Commit=000000000",
		}, " ")),
		"./main.go",
	)
	cmd.Dir = filepath.Dir(b.mainPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", b.operatingSystem),
		fmt.Sprintf("GOARCH=%s", b.procArchitecture),
	)
	r, w := io.Pipe()
	cmd.Stdout = w
	cmd.Stderr = w
	var buf []byte
	var bufMu sync.Mutex
	go func() {
		bufMu.Lock()
		defer bufMu.Unlock()
		buf, _ = io.ReadAll(r)
	}()

	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	err := cmd.Run()
	w.Close()
	if err != nil {
		bufMu.Lock()
		defer bufMu.Unlock()
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("failed to build: %s: %s", err.Error(), string(buf))
	}

	f, err := os.Open(b.outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to get hash: reading binary: %s", err.Error())
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to get hash: %s", err.Error())
	}

	err = b.updateHash()
	if err != nil {
		return "", err
	}
	return b.hash, err
}
