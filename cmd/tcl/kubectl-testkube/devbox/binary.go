// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

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
)

type binaryObj struct {
	lastHash     string
	outputPath   string
	mainFilePath string
	os           string
	arch         string
	mu           sync.Mutex
}

func NewBinary(mainFilePath, outputPath, os, arch string) *binaryObj {
	return &binaryObj{
		mainFilePath: mainFilePath,
		outputPath:   outputPath,
		os:           os,
		arch:         arch,
	}
}

func (b *binaryObj) Hash() string {
	return b.lastHash
}

func (b *binaryObj) Path() string {
	return b.outputPath
}

func (b *binaryObj) Build(ctx context.Context) (hash string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	cmd := exec.Command(
		"go", "build",
		"-o", b.outputPath,
		fmt.Sprintf("-ldflags=%s", strings.Join([]string{
			"-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=",
			"-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=",
			"-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=",
			"-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=",
			"-X github.com/kubeshop/testkube/internal/pkg/api.Version=dev",
			"-X github.com/kubeshop/testkube/internal/pkg/api.Commit=000000000",
		}, " ")),
		"./main.go",
	)
	cmd.Dir = filepath.Dir(b.mainFilePath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", b.os),
		fmt.Sprintf("GOARCH=%s", b.arch),
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

	if err = cmd.Run(); err != nil {
		w.Close()
		bufMu.Lock()
		defer bufMu.Unlock()
		return "", fmt.Errorf("failed to build: %s: %s", err.Error(), string(buf))
	}
	w.Close()

	f, err := os.Open(b.outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to get hash: reading binary: %s", err.Error())
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to get hash: %s", err.Error())
	}

	b.lastHash = fmt.Sprintf("%x", h.Sum(nil))
	return b.lastHash, nil
}
