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
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
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
					Image:           env.Config().Images.Toolkit,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         common.Ptr([]string{"/toolkit", "transfer"}),
					Env: []corev1.EnvVar{
						{Name: "TK_NS", Value: env.Namespace()},
						{Name: "TK_REF", Value: env.Ref()},
						stage.BypassToolkitCheck,
					},
					Args: &result,
				},
			},
		},
	}, nil
}

func CreateExecutionMachine(prefix string, index int64) (string, expressions.Machine) {
	id := fmt.Sprintf("%s-%s%d", env.ExecutionId(), prefix, index)
	fsPrefix := fmt.Sprintf("%s/%s%d", env.Ref(), prefix, index+1)
	if env.Config().Execution.FSPrefix != "" {
		fsPrefix = fmt.Sprintf("%s/%s", env.Config().Execution.FSPrefix, fsPrefix)
	}
	return id, expressions.NewMachine().
		Register("workflow", map[string]string{
			"name": env.WorkflowName(),
		}).
		Register("resource", map[string]string{
			"root":     env.ExecutionId(),
			"id":       id,
			"fsPrefix": fsPrefix,
		}).
		Register("execution", map[string]interface{}{
			"id":              env.ExecutionId(),
			"name":            env.ExecutionName(),
			"number":          env.ExecutionNumber(),
			"scheduledAt":     env.ExecutionScheduledAt().UTC().Format(constants.RFC3339Millis),
			"disableWebhooks": env.ExecutionDisableWebhooks(),
		})
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
		err = storage.SaveStream(filePath, ctrl.Logs(ctx, false))
		ctrl.StopController()
	}
	return filePath, err
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
	dashboardUrl := env.Config().System.DashboardUrl
	if env.Config().Cloud.ApiKey != "" {
		dashboardUrl = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard",
			env.Config().Cloud.UiUrl, env.Config().Cloud.OrgId, env.Config().Cloud.EnvId)
	}
	return expressions.CombinedMachines(
		data.GetBaseTestWorkflowMachine(),
		expressions.NewMachine().RegisterStringMap("internal", map[string]string{
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

			"serviceaccount.default": env.Config().System.DefaultServiceAccount,

			"cloud.enabled":         strconv.FormatBool(env.Config().Cloud.ApiKey != ""),
			"cloud.api.key":         env.Config().Cloud.ApiKey,
			"cloud.api.tlsInsecure": strconv.FormatBool(env.Config().Cloud.TlsInsecure),
			"cloud.api.skipVerify":  strconv.FormatBool(env.Config().Cloud.SkipVerify),
			"cloud.api.url":         env.Config().Cloud.Url,
			"cloud.ui.url":          env.Config().Cloud.UiUrl,
			"cloud.api.orgId":       env.Config().Cloud.OrgId,
			"cloud.api.envId":       env.Config().Cloud.EnvId,

			"dashboard.url":   env.Config().System.DashboardUrl,
			"api.url":         env.Config().System.ApiUrl,
			"namespace":       env.Namespace(),
			"defaultRegistry": env.Config().System.DefaultRegistry,
			"clusterId":       env.Config().System.ClusterID,
			"cdeventsTarget":  env.Config().System.CDEventsTarget,

			"images.defaultRegistry":     env.Config().System.DefaultRegistry,
			"images.init":                env.Config().Images.Init,
			"images.toolkit":             env.Config().Images.Toolkit,
			"images.persistence.enabled": strconv.FormatBool(env.Config().Images.InspectorPersistenceEnabled),
			"images.persistence.key":     env.Config().Images.InspectorPersistenceCacheKey,
			"images.cache.ttl":           env.Config().Images.ImageCredentialsCacheTTL.String(),
		}).
			Register("dashboard", map[string]string{
				"url": dashboardUrl,
			}).
			Register("organization", map[string]string{
				"id": env.Config().Cloud.OrgId,
			}).
			Register("environment", map[string]string{
				"id": env.Config().Cloud.EnvId,
			}),
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
