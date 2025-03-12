package testworkflowconfig

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func CreateExecutionMachine(cfg *ExecutionConfig) expressions.Machine {
	return expressions.NewMachine().
		Register("execution", map[string]interface{}{
			"id":              cfg.Id,
			"groupId":         cfg.GroupId,
			"name":            cfg.Name,
			"number":          cfg.Number,
			"scheduledAt":     cfg.ScheduledAt,
			"disableWebhooks": cfg.DisableWebhooks,
			"tags":            cfg.Tags,
		}).
		RegisterStringMap("organization", map[string]string{
			"id": cfg.OrganizationId,
		}).
		RegisterStringMap("environment", map[string]string{
			"id": cfg.EnvironmentId,
		})
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
