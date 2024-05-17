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
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	common2 "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
)

func ProcessTransfer(transferSrv transfer.Server, transfer []testworkflowsv1.StepParallelTransfer, machines ...expressionstcl.Machine) ([]testworkflowsv1.ContentTarball, error) {
	if len(transfer) == 0 {
		return nil, nil
	}
	result := make([]testworkflowsv1.ContentTarball, 0, len(transfer))
	for ti, t := range transfer {
		// Parse 'from' clause
		from, err := expressionstcl.EvalTemplate(t.From, machines...)
		if err != nil {
			return nil, errors.Wrapf(err, "%d.from", ti)
		}

		// Parse 'to' clause
		to := from
		if t.To != "" {
			to, err = expressionstcl.EvalTemplate(t.To, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "%d.to", ti)
			}
		}

		// Parse 'files' clause
		patterns := []string{"**/*"}
		if t.Files != nil && !t.Files.Dynamic {
			patterns = t.Files.Static
		} else if t.Files != nil && t.Files.Dynamic {
			patternsExpr, err := expressionstcl.EvalExpression(t.Files.Expression, machines...)
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

func ProcessFetch(transferSrv transfer.Server, fetch []testworkflowsv1.StepParallelFetch, machines ...expressionstcl.Machine) (*testworkflowsv1.Step, error) {
	if len(fetch) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(fetch))
	for ti, t := range fetch {
		// Parse 'from' clause
		from, err := expressionstcl.EvalTemplate(t.From, machines...)
		if err != nil {
			return nil, errors.Wrapf(err, "%d.from", ti)
		}

		// Parse 'to' clause
		to := from
		if t.To != "" {
			to, err = expressionstcl.EvalTemplate(t.To, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "%d.to", ti)
			}
		}

		// Parse 'files' clause
		patterns := []string{"**/*"}
		if t.Files != nil && !t.Files.Dynamic {
			patterns = t.Files.Static
		} else if t.Files != nil && t.Files.Dynamic {
			patternsExpr, err := expressionstcl.EvalExpression(t.Files.Expression, machines...)
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
					Image:           env.Config().Images.Toolkit,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         common.Ptr([]string{"/toolkit", "transfer"}),
					Env: []corev1.EnvVar{
						{Name: "TK_NS", Value: env.Namespace()},
						{Name: "TK_REF", Value: env.Ref()},
					},
					Args: &result,
				},
			},
		},
	}, nil
}

func CreateExecutionMachine(prefix string, index int64) (string, expressionstcl.Machine) {
	id := fmt.Sprintf("%s-%s%d", env.ExecutionId(), prefix, index)
	fsPrefix := fmt.Sprintf("%s/%s%d", env.Ref(), prefix, index+1)
	if env.Config().Execution.FSPrefix != "" {
		fsPrefix = fmt.Sprintf("%s/%s", env.Config().Execution.FSPrefix, fsPrefix)
	}
	// TODO(emil): register path for custom ca cert
	// TODO(emil): register new values here for istio configs
	return id, expressionstcl.NewMachine().
		Register("execution.id", env.ExecutionId()).
		Register("resource.rootId", env.ExecutionId()).
		Register("resource.id", id).
		Register("resource.fsPrefix", fsPrefix).
		Register("workflow.name", env.WorkflowName())
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

func SaveLogs(ctx context.Context, clientSet kubernetes.Interface, storage artifacts.InternalArtifactStorage, namespace, id, prefix string, index int64) (string, error) {
	filePath := fmt.Sprintf("logs/%s%d.log", prefix, index)
	ctrl, err := testworkflowcontroller.New(ctx, clientSet, namespace, id, time.Time{}, testworkflowcontroller.ControllerOptions{
		Timeout: ControllerTimeout,
	})
	if err == nil {
		err = storage.SaveStream(filePath, ctrl.Logs(ctx))
	}
	return filePath, err
}

func CreateLogger(name, description string, index, count int64) func(...string) {
	label := common2.InstanceLabel(name, index, count)
	if description != "" {
		label += " (" + description + ")"
	}
	return func(s ...string) {
		fmt.Printf("%s: %s\n", label, strings.Join(s, ": "))
	}
}

func CreateBaseMachine() expressionstcl.Machine {
	return expressionstcl.CombinedMachines(
		data.GetBaseTestWorkflowMachine(),
		expressionstcl.NewMachine().RegisterStringMap("internal", map[string]string{
			"storage.url":        env.Config().ObjectStorage.Endpoint,
			"storage.accessKey":  env.Config().ObjectStorage.AccessKeyID,
			"storage.secretKey":  env.Config().ObjectStorage.SecretAccessKey,
			"storage.region":     env.Config().ObjectStorage.Region,
			"storage.bucket":     env.Config().ObjectStorage.Bucket,
			"storage.token":      env.Config().ObjectStorage.Token,
			"storage.ssl":        strconv.FormatBool(env.Config().ObjectStorage.Ssl),
			"storage.skipVerify": strconv.FormatBool(env.Config().ObjectStorage.SkipVerify),
			"storage.certFile":   env.Config().ObjectStorage.CertFile,
			"storage.keyFile":    env.Config().ObjectStorage.KeyFile,
			"storage.caFile":     env.Config().ObjectStorage.CAFile,

			"cloud.enabled":         strconv.FormatBool(env.Config().Cloud.ApiKey != ""),
			"cloud.api.key":         env.Config().Cloud.ApiKey,
			"cloud.api.tlsInsecure": strconv.FormatBool(env.Config().Cloud.TlsInsecure),
			"cloud.api.skipVerify":  strconv.FormatBool(env.Config().Cloud.SkipVerify),
			"cloud.api.url":         env.Config().Cloud.Url,

			"dashboard.url":   env.Config().System.DashboardUrl,
			"api.url":         env.Config().System.ApiUrl,
			"namespace":       env.Namespace(),
			"defaultRegistry": env.Config().System.DefaultRegistry,

			"images.init":                env.Config().Images.Init,
			"images.toolkit":             env.Config().Images.Toolkit,
			"images.persistence.enabled": strconv.FormatBool(env.Config().Images.InspectorPersistenceEnabled),
			"images.persistence.key":     env.Config().Images.InspectorPersistenceCacheKey,
		}),
	)
}

func CreateResultMachine(result testkube.TestWorkflowResult) expressionstcl.Machine {
	status := "queued"
	if result.Status != nil {
		if *result.Status == testkube.PASSED_TestWorkflowStatus {
			status = ""
		} else {
			status = string(*result.Status)
		}
	}
	return expressionstcl.NewMachine().
		Register("status", status).
		Register("always", true).
		Register("never", false).
		Register("failed", status != "").
		Register("error", status != "").
		Register("passed", status == "").
		Register("success", status == "")
}

func EvalLogCondition(condition string, result testkube.TestWorkflowResult, machines ...expressionstcl.Machine) (bool, error) {
	expr, err := expressionstcl.EvalExpression(condition, append([]expressionstcl.Machine{CreateResultMachine(result)}, machines...)...)
	if err != nil {
		return false, errors.Wrapf(err, "invalid expression for logs condition: %s", condition)
	}
	return expr.BoolValue()
}
