// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
// 	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MockRegistry is a simple in-memory implementation of a worker registry for testing
type MockRegistry struct {
	workers map[int64]*MockWorker
	count   int64
}

type MockWorker struct {
	status  *testkube.TestWorkflowStatus
	address string
}

func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		workers: make(map[int64]*MockWorker),
	}
}

func (r *MockRegistry) SetStatus(index int64, status *testkube.TestWorkflowStatus) {
	if _, exists := r.workers[index]; !exists {
		r.workers[index] = &MockWorker{}
		r.count++
	}
	r.workers[index].status = status
}

func (r *MockRegistry) SetAddress(index int64, address string) {
	if _, exists := r.workers[index]; !exists {
		r.workers[index] = &MockWorker{}
		r.count++
	}
	r.workers[index].address = address
}

func (r *MockRegistry) Count() int64 {
	return r.count
}

func (r *MockRegistry) AllPaused() bool {
	if r.count == 0 {
		return false
	}
	for _, w := range r.workers {
		if w.status == nil || *w.status != testkube.PAUSED_TestWorkflowStatus {
			return false
		}
	}
	return true
}

func (r *MockRegistry) Destroy(index int64) {
	if _, exists := r.workers[index]; exists {
		delete(r.workers, index)
		r.count--
	}
}

func (r *MockRegistry) Indexes() []int64 {
	indexes := make([]int64, 0, len(r.workers))
	for idx := range r.workers {
		indexes = append(indexes, idx)
	}
	return indexes
}

func TestParallelSpecParsing(t *testing.T) {
	parser := &ParallelSpecParser{}

	t.Run("input-formats", func(t *testing.T) {
		validSpec := &testworkflowsv1.StepParallel{
			StepOperations: testworkflowsv1.StepOperations{
				Shell: `echo "test"`,
			},
		}
		jsonBytes, _ := json.Marshal(validSpec)
		base64Encoded := base64.StdEncoding.EncodeToString(jsonBytes)

		testCases := []struct {
			name        string
			args        []string
			shouldError bool
			validate    func(t *testing.T, result *testworkflowsv1.StepParallel)
		}{
			{
				name:        "parses raw JSON",
				args:        []string{string(jsonBytes)},
				shouldError: false,
				validate: func(t *testing.T, result *testworkflowsv1.StepParallel) {
					assert.Equal(t, validSpec.Shell, result.Shell)
				},
			},
			{
				name:        "parses base64 encoded JSON",
				args:        []string{"--base64", base64Encoded},
				shouldError: false,
				validate: func(t *testing.T, result *testworkflowsv1.StepParallel) {
					assert.Equal(t, validSpec.Shell, result.Shell)
				},
			},
			{
				name:        "handles escaped quotes in JSON",
				args:        []string{`{"shell":"echo \"test\""}`},
				shouldError: false,
				validate: func(t *testing.T, result *testworkflowsv1.StepParallel) {
					assert.Equal(t, `echo "test"`, result.Shell)
				},
			},
			{
				name:        "rejects invalid base64",
				args:        []string{"--base64", "not-valid-base64!@#"},
				shouldError: true,
			},
			{
				name:        "rejects invalid JSON",
				args:        []string{"{invalid json}"},
				shouldError: true,
			},
			{
				name:        "rejects empty args",
				args:        []string{},
				shouldError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Extract base64 flag from args if present
				base64Encoded := len(tc.args) >= 2 && tc.args[0] == "--base64"
				specContent := ""
				if base64Encoded && len(tc.args) > 1 {
					specContent = tc.args[1]
				} else if len(tc.args) > 0 {
					specContent = tc.args[0]
				}
				result, err := parser.ParseSpec(specContent, base64Encoded)

				if tc.shouldError {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					if tc.validate != nil {
						tc.validate(t, result)
					}
				}
			})
		}
	})

	t.Run("spec-normalization", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    func() *testworkflowsv1.StepParallel
			validate func(t *testing.T, spec *testworkflowsv1.StepParallel)
		}{
			{
				name: "injects short syntax as first step",
				input: func() *testworkflowsv1.StepParallel {
					return &testworkflowsv1.StepParallel{
						StepOperations: testworkflowsv1.StepOperations{
							Shell: `echo "inline"`,
						},
						Steps: []testworkflowsv1.Step{
							{StepOperations: testworkflowsv1.StepOperations{Shell: `echo "step"`}},
						},
					}
				},
				validate: func(t *testing.T, spec *testworkflowsv1.StepParallel) {
					require.Len(t, spec.Steps, 2)
					assert.Equal(t, `echo "inline"`, spec.Steps[0].Shell)
					assert.Equal(t, `echo "step"`, spec.Steps[1].Shell)
					assert.Empty(t, spec.Shell) // Should be cleared
				},
			},
			{
				name: "adds default service account",
				input: func() *testworkflowsv1.StepParallel {
					return &testworkflowsv1.StepParallel{}
				},
				validate: func(t *testing.T, spec *testworkflowsv1.StepParallel) {
					require.NotNil(t, spec.Pod)
					assert.Equal(t, "{{internal.serviceaccount.default}}", spec.Pod.ServiceAccountName)
				},
			},
			{
				name: "preserves StepExecuteStrategy",
				input: func() *testworkflowsv1.StepParallel {
					return &testworkflowsv1.StepParallel{
						StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
							Count:    common.Ptr(intstr.FromInt(5)),
							MaxCount: common.Ptr(intstr.FromInt(10)),
							Matrix: map[string]testworkflowsv1.DynamicList{
								"version": {Static: []any{"1", "2"}},
							},
						},
					}
				},
				validate: func(t *testing.T, spec *testworkflowsv1.StepParallel) {
					assert.Equal(t, 5, spec.Count.IntValue())
					assert.Equal(t, 10, spec.MaxCount.IntValue())
					assert.Contains(t, spec.Matrix, "version")
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				spec := tc.input()
				parser.NormalizeParallelSpec(spec)
				tc.validate(t, spec)
			})
		}
	})
}

func TestWorkerOrchestration(t *testing.T) {
	t.Run("registry-operations", func(t *testing.T) {
		registry := NewMockRegistry()

		// Test basic operations
		registry.SetStatus(0, nil)
		assert.Equal(t, int64(1), registry.Count())

		status := testkube.RUNNING_TestWorkflowStatus
		registry.SetStatus(0, &status)
		registry.SetAddress(0, "10.0.0.1")

		registry.SetStatus(1, &status)
		assert.Equal(t, int64(2), registry.Count())

		// Test AllPaused
		assert.False(t, registry.AllPaused())

		pausedStatus := testkube.PAUSED_TestWorkflowStatus
		registry.SetStatus(0, &pausedStatus)
		registry.SetStatus(1, &pausedStatus)
		assert.True(t, registry.AllPaused())

		// Test cleanup
		registry.Destroy(0)
		assert.Equal(t, int64(1), registry.Count())
	})

	t.Run("orchestrator-lifecycle", func(t *testing.T) {
		registry := NewMockRegistry()
		updates := make(chan Update, 10)
		cfg := &config.ConfigV2{}
		orchestrator := NewResumeOrchestrator(registry, updates, []string{"ns1"}, []string{"worker-1"}, cfg)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})

		go func() {
			orchestrator.Start(ctx)
			close(done)
		}()

		// Send status update
		status := testkube.RUNNING_TestWorkflowStatus
		updates <- Update{
			index:  0,
			result: &testkube.TestWorkflowResult{Status: &status},
		}

		// Wait for processing
		require.Eventually(t, func() bool {
			return registry.Count() == 1
		}, 100*time.Millisecond, 5*time.Millisecond)

		// Test cleanup on done
		updates <- Update{index: 0, done: true}
		require.Eventually(t, func() bool {
			return registry.Count() == 0
		}, 100*time.Millisecond, 5*time.Millisecond)

		// Test graceful shutdown
		cancel()
		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("orchestrator did not shut down gracefully")
		}
	})
}

func TestTransferServer(t *testing.T) {
	// Helper for server readiness checks
	waitForServer := func(t *testing.T, url string, method string, body io.Reader) (*http.Response, error) {
		t.Helper()

		var resp *http.Response
		var err error

		for i := 0; i < 5; i++ {
			switch method {
			case "GET":
				resp, err = http.Get(url)
			case "POST":
				resp, err = http.Post(url, "application/octet-stream", body)
			}

			if err == nil {
				return resp, nil
			}

			// Exponential backoff: 1ms, 2ms, 4ms, 8ms, 16ms
			time.Sleep(time.Duration(1<<uint(i)) * time.Millisecond)
		}

		return nil, err
	}

	t.Run("server-lifecycle", func(t *testing.T) {
		// Test empty server doesn't start
		server := transfer.NewServer("/tmp", "127.0.0.1", 0)
		err := StartTransferServer(server)
		assert.NoError(t, err, "Empty server should not start")

		// Test server with files
		tmpDir := t.TempDir()
		testFile := fmt.Sprintf("%s/test.txt", tmpDir)
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		server = transfer.NewServer(tmpDir, "127.0.0.1", 0)
		_, err = server.Include("/test/dir", []string{"test.txt"})
		require.NoError(t, err)

		err = StartTransferServer(server)
		assert.NoError(t, err)
		assert.Greater(t, server.Count(), 0)
	})

	t.Run("file-operations", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test file
		testFile := fmt.Sprintf("%s/test.txt", tmpDir)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		server := transfer.NewServer(tmpDir, "127.0.0.1", 58901)

		// Test download
		entry, err := server.Include("/test/dir", []string{"test.txt"})
		require.NoError(t, err)

		err = StartTransferServer(server)
		require.NoError(t, err)

		resp, err := waitForServer(t, entry.Url, "GET", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.NotEmpty(t, body)
	})

	t.Run("error-handling", func(t *testing.T) {
		// Test invalid port
		server := transfer.NewServer("/tmp", "127.0.0.1", -1)
		server.Include("/test", []string{"dummy.txt"})

		err := StartTransferServer(server)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start transfer server")
	})
}

func TestParallelExecution(t *testing.T) {
	minimalTkConfig := `{
		"namespace": "test",
		"resource": {"id": "test", "root": "/tmp", "fsPrefix": "test"},
		"workflow": {"name": "test", "labels": {}},
		"execution": {"id": "test", "organizationId": "test", "environmentId": "test", "pvcNames": {}},
		"controlPlane": {"url": "http://localhost:8088"},
		"worker": {"namespace": "test", "connection": {"url": "http://localhost:8088"}}
	}`
	t.Setenv("TK_CFG", minimalTkConfig)

	cfg, err := config.LoadConfigV2()
	require.NoError(t, err, "Failed to load config")

	// Create NoOp storage for tests
	storage, err := artifacts.InternalStorageWithProvider(&artifacts.NoOpStorageProvider{}, cfg)
	require.NoError(t, err)

	t.Run("execution-flow", func(t *testing.T) {
		testCases := []struct {
			name          string
			args          []string
			shouldError   bool
			errorContains string
		}{
			{
				name:          "invalid spec",
				args:          []string{"{invalid json}"},
				shouldError:   true,
				errorContains: "parsing parallel spec",
			},
			{
				name: "zero workers",
				args: func() []string {
					spec := &testworkflowsv1.StepParallel{
						StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
							Count: common.Ptr(intstr.FromInt(0)),
						},
					}
					jsonBytes, _ := json.Marshal(spec)
					return []string{string(jsonBytes)}
				}(),
				shouldError: false, // Current behavior - succeeds with no work
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()

				specContent := ""
				if len(tc.args) > 0 {
					specContent = tc.args[0]
				}
				opts := &ParallelOptions{Storage: storage}
				err := RunParallelWithOptions(ctx, specContent, cfg, false, opts)

				if tc.shouldError {
					assert.Error(t, err)
					if tc.errorContains != "" {
						assert.Contains(t, err.Error(), tc.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("backwards-compatibility", func(t *testing.T) {
		// Test both raw JSON and base64 work
		spec := &testworkflowsv1.StepParallel{
			StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
				Count: common.Ptr(intstr.FromInt(0)),
			},
		}
		jsonBytes, _ := json.Marshal(spec)

		opts := &ParallelOptions{Storage: storage}

		// Raw JSON
		err = RunParallelWithOptions(context.Background(), string(jsonBytes), cfg, false, opts)
		assert.NoError(t, err)

		// Base64
		encoded := base64.StdEncoding.EncodeToString(jsonBytes)
		err = RunParallelWithOptions(context.Background(), encoded, cfg, true, opts)
		assert.NoError(t, err)
	})
}
