package testworkflowconfig

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func CreateExecutionMachine(cfg *ExecutionConfig) expressions.Machine {
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

	return expressions.NewMachine().
		Register("execution", execution).
		RegisterStringMap("organization", map[string]string{
			"id": cfg.OrganizationId,
		}).
		RegisterStringMap("environment", map[string]string{
			"id": cfg.EnvironmentId,
		})
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
