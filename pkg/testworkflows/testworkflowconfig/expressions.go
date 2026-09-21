package testworkflowconfig

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
)

// CreateExecutionMachine exposes execution.* to expressions resolved inside the
// pod, where every value is the running execution's own.
func CreateExecutionMachine(cfg *ExecutionConfig) expressions.Machine {
	return createExecutionMachine(cfg, true)
}

// CreateSchedulingExecutionMachine is CreateExecutionMachine for the pass that
// resolves the workflow spec before it is scheduled, and it deliberately leaves
// execution.lineage alone.
//
// That pass's output is stored as TestWorkflowExecution.ResolvedWorkflow, and a
// rerun replays that snapshot verbatim unless it is asked for the latest
// definition. Whatever this machine substitutes is therefore frozen at the
// values of the run that produced the snapshot - which is harmless for an id or
// a number, since a reader of those wants to know which run it descends from,
// but wrong for lineage: baking the original's "no base, attempt 1" into the
// snapshot makes every rerun of it take the original branch, so a workflow that
// selects failed cases behind `execution.lineage.baseId != ""` would run the
// whole suite forever and never reach execution("rerun").
//
// Not registering the accessor is what keeps `{{ execution.lineage.* }}`
// dynamic rather than empty: an accessor nothing matches resolves to itself, so
// the expression survives into the snapshot and is resolved per execution in the
// pod, where data.ExecutionMachine() supplies that execution's own lineage.
// Step conditions reach it too - ResolveCondition evaluates them through
// data.Expression, which appends the same machine.
//
// The cost is that lineage cannot be used where a literal is needed before the
// pod exists, an image tag or a resource limit, because nothing resolves it at
// that point.
func CreateSchedulingExecutionMachine(cfg *ExecutionConfig) expressions.Machine {
	return createExecutionMachine(cfg, false)
}

func createExecutionMachine(cfg *ExecutionConfig, withLineage bool) expressions.Machine {
	execution := map[string]interface{}{
		"id":              cfg.Id,
		"groupId":         cfg.GroupId,
		"name":            cfg.Name,
		"number":          cfg.Number,
		"scheduledAt":     cfg.ScheduledAt,
		"disableWebhooks": cfg.DisableWebhooks,
		"tags":            cfg.Tags,
	}
	execution["runningContext"] = buildRunningContext(cfg.RunningContext)
	if withLineage {
		execution["lineage"] = buildLineage(cfg.Lineage, cfg.Id)
	}

	return expressions.NewMachine().
		Register("execution", execution).
		RegisterStringMap("organization", map[string]string{
			"id": cfg.OrganizationId,
		}).
		RegisterStringMap("environment", map[string]string{
			"id": cfg.EnvironmentId,
		})
}

// buildLineage exposes execution.lineage.* to the workflow spec.
//
// An execution with no lineage recorded resolves to the original-run defaults
// rather than to nothing, so that {{ execution.lineage.attempt }} is usable in
// every workflow instead of only in reruns.
func buildLineage(cfg *LineageConfig, executionId string) map[string]interface{} {
	// The defaults have to match TestWorkflowExecution.EffectiveLineage() field
	// by field, or an execution recorded before lineage existed reports one
	// lineage through the API and a different one to its own expressions. An
	// original run is its own root, so rootId falls back to this execution id
	// rather than to the empty string.
	var lineage LineageConfig
	if cfg != nil {
		lineage = *cfg
	}
	if lineage.RootId == "" {
		lineage.RootId = executionId
	}
	if lineage.Attempt == 0 {
		lineage.Attempt = 1
	}
	return map[string]interface{}{
		"baseId":  lineage.BaseId,
		"rootId":  lineage.RootId,
		"attempt": lineage.Attempt,
	}
}

func buildRunningContext(rc *testkube.TestWorkflowRunningContext) map[string]interface{} {
	actor := map[string]interface{}{}
	interfaceMap := map[string]interface{}{}
	if rc == nil {
		return map[string]interface{}{
			"actor":     actor,
			"interface": interfaceMap,
		}
	}

	if rc.Actor != nil {
		actor = map[string]interface{}{
			"name":               rc.Actor.Name,
			"email":              rc.Actor.Email,
			"executionId":        rc.Actor.ExecutionId,
			"executionPath":      rc.Actor.ExecutionPath,
			"executionReference": rc.Actor.ExecutionReference,
		}
		if rc.Actor.Type_ != nil {
			actor["type"] = string(*rc.Actor.Type_)
		}
	}

	if rc.Interface_ != nil {
		interfaceMap = map[string]interface{}{
			"name": rc.Interface_.Name,
		}
		if rc.Interface_.Type_ != nil {
			interfaceMap["type"] = string(*rc.Interface_.Type_)
		}
	}

	return map[string]interface{}{
		"actor":     actor,
		"interface": interfaceMap,
	}
}

func CreateCloudMachine(cfg *ControlPlaneConfig, orgSlug, envSlug string) expressions.Machine {
	dashboardUrl := cfg.DashboardUrl
	if dashboardUrl != "" && orgSlug != "" && envSlug != "" {
		dashboardUrl = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard", cfg.DashboardUrl, orgSlug, envSlug)
	}
	return expressions.NewMachine().
		RegisterMap("internal", map[string]interface{}{
			"dashboard.url":  dashboardUrl,
			"cdeventsTarget": cfg.CDEventsTarget,
		}).
		RegisterStringMap("dashboard", map[string]string{
			"url": cfg.DashboardUrl,
		})
}

func CreateWorkflowMachine(cfg *WorkflowConfig) expressions.Machine {
	escapedLabels := make(map[string]string)
	for key, value := range cfg.Labels {
		escapedLabels[expressions.EscapeLabelKeyForVarName(key)] = value
	}
	return expressions.NewMachine().
		RegisterMap("workflow", map[string]interface{}{
			"name":   cfg.Name,
			"labels": cfg.Labels,
		}).
		RegisterStringMap("labels", escapedLabels)
}

func CreateResourceMachine(cfg *ResourceConfig) expressions.Machine {
	return expressions.NewMachine().Register("resource", map[string]string{
		"id":       cfg.Id,
		"root":     cfg.RootId,
		"fsPrefix": cfg.FsPrefix,
	})
}

func CreateWorkerMachine(cfg *WorkerConfig) expressions.Machine {
	machine := expressions.NewMachine().
		RegisterMap("internal", map[string]interface{}{
			"serviceaccount.default": cfg.DefaultServiceAccount,
			"namespace":              cfg.Namespace,
			"clusterId":              cfg.ClusterID,

			"cloud.enabled":         cfg.Connection.ApiKey != "",
			"cloud.api.key":         cfg.Connection.ApiKey,
			"cloud.api.tlsInsecure": cfg.Connection.TlsInsecure,
			"cloud.api.skipVerify":  cfg.Connection.SkipVerify,
			"cloud.api.url":         cfg.Connection.Url,

			"images.defaultRegistry":     cfg.DefaultRegistry,
			"images.insecureRegistries":  cfg.InsecureRegistries,
			"images.init":                cfg.InitImage,
			"images.toolkit":             cfg.ToolkitImage,
			"images.persistence.enabled": cfg.ImageInspectorPersistenceEnabled,
			"images.persistence.key":     cfg.ImageInspectorPersistenceCacheKey,
			"images.cache.ttl":           cfg.ImageInspectorPersistenceCacheTTL,

			"api.url": cfg.Connection.LocalApiUrl, // TODO: Delete
		})
	return expressions.CombinedMachines(machine)
}

func CreatePvcMachine(pvcNames map[string]string) expressions.Machine {
	pvcMap := make(map[string]string)
	for name, pvcName := range pvcNames {
		pvcMap[name+".name"] = pvcName
	}

	return expressions.NewMachine().RegisterStringMap("pvcs", pvcMap)
}
