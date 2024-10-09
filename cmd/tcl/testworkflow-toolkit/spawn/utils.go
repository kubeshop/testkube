// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	LogsRetryOnFailureDelay = 300 * time.Millisecond
	LogsRetryMaxAttempts    = 5
)

func MapDynamicListToStringList(list []interface{}) []string {
	result := make([]string, len(list))
	for i := range list {
		if v, ok := list[i].(string); ok {
			result[i] = v
		} else {
			b, _ := json.Marshal(list[i])
			result[i] = string(b)
		}
	}
	return result
}

func ProcessTransfer(transferSrv transfer.Server, transfer []testworkflowsv1.StepParallelTransfer, machines ...expressions.Machine) ([]testworkflowsv1.ContentTarball, error) {
	if len(transfer) == 0 {
		return nil, nil
	}
	result := make([]testworkflowsv1.ContentTarball, 0, len(transfer))
	for ti, t := range transfer {
		// Parse 'from' clause
		from, err := expressions.EvalTemplate(t.From, machines...)
		if err != nil {
			return nil, errors.Wrapf(err, "%d.from", ti)
		}

		// Parse 'to' clause
		to := from
		if t.To != "" {
			to, err = expressions.EvalTemplate(t.To, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "%d.to", ti)
			}
		}

		// Parse 'files' clause
		patterns := []string{"**/*"}
		if t.Files != nil && !t.Files.Dynamic {
			patterns = MapDynamicListToStringList(t.Files.Static)
		} else if t.Files != nil && t.Files.Dynamic {
			patternsExpr, err := expressions.EvalExpression(t.Files.Expression, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "%d.files", ti)
			}
			patternsList, err := patternsExpr.Static().SliceValue()
			if err != nil {
				return nil, errors.Wrapf(err, "%d.files", ti)
			}
			patterns = make([]string, len(patternsList))
			for pi, p := range patternsList {
				if s, ok := p.(string); ok {
					patterns[pi] = s
				} else {
					p, err := json.Marshal(s)
					if err != nil {
						return nil, errors.Wrapf(err, "%d.files.%d", ti, pi)
					}
					patterns[pi] = string(p)
				}
			}
		}

		entry, err := transferSrv.Include(from, patterns)
		if err != nil {
			return nil, errors.Wrapf(err, "%d", ti)
		}
		result = append(result, testworkflowsv1.ContentTarball{Url: entry.Url, Path: to, Mount: t.Mount})
	}
	return result, nil
}

func ProcessFetch(transferSrv transfer.Server, fetch []testworkflowsv1.StepParallelFetch, machines ...expressions.Machine) (*testworkflowsv1.Step, error) {
	if len(fetch) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(fetch))
	for ti, t := range fetch {
		// Parse 'from' clause
		from, err := expressions.EvalTemplate(t.From, machines...)
		if err != nil {
			return nil, errors.Wrapf(err, "%d.from", ti)
		}

		// Parse 'to' clause
		to := from
		if t.To != "" {
			to, err = expressions.EvalTemplate(t.To, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "%d.to", ti)
			}
		}

		// Parse 'files' clause
		patterns := []string{"**/*"}
		if t.Files != nil && !t.Files.Dynamic {
			patterns = MapDynamicListToStringList(t.Files.Static)
		} else if t.Files != nil && t.Files.Dynamic {
			patternsExpr, err := expressions.EvalExpression(t.Files.Expression, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "%d.files", ti)
			}
			patternsList, err := patternsExpr.Static().SliceValue()
			if err != nil {
				return nil, errors.Wrapf(err, "%d.files", ti)
			}
			patterns = make([]string, len(patternsList))
			for pi, p := range patternsList {
				if s, ok := p.(string); ok {
					patterns[pi] = s
				} else {
					p, err := json.Marshal(s)
					if err != nil {
						return nil, errors.Wrapf(err, "%d.files.%d", ti, pi)
					}
					patterns[pi] = string(p)
				}
			}
		}

		req := transferSrv.Request(to)
		result = append(result, fmt.Sprintf("%s:%s=%s", from, strings.Join(patterns, ","), req.Url))
	}

	return &testworkflowsv1.Step{
		StepMeta: testworkflowsv1.StepMeta{
			Name:      "Save the files",
			Condition: "always",
		},
		StepOperations: testworkflowsv1.StepOperations{
			Run: &testworkflowsv1.StepRun{
				ContainerConfig: testworkflowsv1.ContainerConfig{
					Image:           config.Config().Worker.ToolkitImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         common.Ptr([]string{constants.DefaultToolkitPath, "transfer"}),
					Env: []corev1.EnvVar{
						{Name: "TK_NS", Value: config.Namespace()},
						{Name: "TK_REF", Value: config.Ref()},
						stage.BypassToolkitCheck,
						stage.BypassPure,
					},
					Args: &result,
				},
			},
		},
	}, nil
}

func CreateResourceConfig(prefix string, index int64) testworkflowconfig.ResourceConfig {
	cfg := config.Config()
	id := fmt.Sprintf("%s-%s%d", cfg.Resource.Id, prefix, index)
	fsPrefix := fmt.Sprintf("%s/%s%d", config.Ref(), prefix, index+1)
	if cfg.Resource.FsPrefix != "" {
		fsPrefix = fmt.Sprintf("%s/%s", cfg.Resource.FsPrefix, fsPrefix)
	}
	return testworkflowconfig.ResourceConfig{
		Id:       id,
		RootId:   cfg.Resource.RootId,
		FsPrefix: fsPrefix,
	}
}

func GetServiceByResourceId(jobName string) (string, int64) {
	regex := regexp.MustCompile(`-(.+?)-(\d+)$`)
	v := regex.FindSubmatch([]byte(jobName))
	if v == nil {
		return "", 0
	}
	index, err := strconv.ParseInt(string(v[2]), 10, 64)
	if err != nil {
		return "", 0
	}
	return string(v[1]), index
}

func ExecuteParallel[T any](run func(int64, *T) bool, items []T, parallelism int64) int64 {
	var wg sync.WaitGroup
	wg.Add(len(items))
	ch := make(chan struct{}, parallelism)
	success := atomic.Int64{}

	// Execute all operations
	for index := range items {
		ch <- struct{}{}
		go func(index int) {
			if run(int64(index), &items[index]) {
				success.Add(1)
			}
			<-ch
			wg.Done()
		}(index)
	}
	wg.Wait()
	return int64(len(items)) - success.Load()
}

func SaveLogsWithController(parentCtx context.Context, storage artifacts.InternalArtifactStorage, ctrl testworkflowcontroller.Controller, prefix string, index int64) (string, error) {
	if ctrl == nil {
		return "", errors.New("cannot control TestWorkflow's execution")
	}

	filePath := fmt.Sprintf("logs/%s%d.log", prefix, index)
	var err error
	for i := 0; i < LogsRetryMaxAttempts; i++ {
		ctx, ctxCancel := context.WithCancel(parentCtx)
		err = storage.SaveStream(filePath, ctrl.Logs(ctx, false))
		ctxCancel()
		if err == nil {
			break
		}
		time.Sleep(LogsRetryOnFailureDelay)
	}

	return filePath, err
}

func SaveLogs(ctx context.Context, clientSet kubernetes.Interface, storage artifacts.InternalArtifactStorage, namespace, id, prefix string, index int64) (string, error) {
	ctrl, err := testworkflowcontroller.New(ctx, clientSet, namespace, id, time.Time{})
	if err != nil {
		return "", err
	}
	defer ctrl.StopController()
	return SaveLogsWithController(ctx, storage, ctrl, prefix, index)
}

func CreateLogger(name, description string, index, count int64) func(...string) {
	label := commontcl.InstanceLabel(name, index, count)
	if description != "" {
		label += " (" + description + ")"
	}
	return func(s ...string) {
		fmt.Printf("%s: %s\n", label, strings.Join(s, ": "))
	}
}

func CreateBaseMachine() expressions.Machine {
	return expressions.CombinedMachines(
		data.GetBaseTestWorkflowMachine(),
		testworkflowconfig.CreateCloudMachine(&config.Config().ControlPlane),
		testworkflowconfig.CreateExecutionMachine(&config.Config().Execution),
		testworkflowconfig.CreateWorkflowMachine(&config.Config().Workflow),
	)
}

func CreateResultMachine(result testkube.TestWorkflowResult) expressions.Machine {
	status := "queued"
	if result.Status != nil {
		if *result.Status == testkube.PASSED_TestWorkflowStatus {
			status = ""
		} else {
			status = string(*result.Status)
		}
	}
	return expressions.NewMachine().
		Register("status", status).
		Register("always", true).
		Register("never", false).
		Register("failed", status != "").
		Register("error", status != "").
		Register("passed", status == "").
		Register("success", status == "")
}

func EvalLogCondition(condition string, result testkube.TestWorkflowResult, machines ...expressions.Machine) (bool, error) {
	expr, err := expressions.EvalExpression(condition, append([]expressions.Machine{CreateResultMachine(result)}, machines...)...)
	if err != nil {
		return false, errors.Wrapf(err, "invalid expression for logs condition: %s", condition)
	}
	return expr.BoolValue()
}
