package testworkflowconfig

import "github.com/kubeshop/testkube/pkg/expressions"

func CreateStorageMachine(cfg *ObjectStorageConfig) expressions.Machine {
	return expressions.NewMachine().
		RegisterMap("internal", map[string]interface{}{
			"storage.url":        cfg.Endpoint,
			"storage.accessKey":  cfg.AccessKeyID,
			"storage.secretKey":  cfg.SecretAccessKey,
			"storage.region":     cfg.Region,
			"storage.bucket":     cfg.Bucket,
			"storage.token":      cfg.Token,
			"storage.ssl":        cfg.Ssl,
			"storage.skipVerify": cfg.SkipVerify,
			"storage.certFile":   cfg.CertFile,
			"storage.keyFile":    cfg.KeyFile,
			"storage.caFile":     cfg.CAFile,
		})
}

func CreateExecutionMachine(cfg *ExecutionConfig) expressions.Machine {
	return expressions.NewMachine().
		Register("execution", map[string]interface{}{
			"id":              cfg.Id,
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

func CreateCloudMachine(cfg *ControlPlaneConfig) expressions.Machine {
	return expressions.NewMachine().
		RegisterMap("internal", map[string]interface{}{
			"dashboard.url":  cfg.DashboardUrl,
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
	return expressions.CombinedMachines(machine, CreateStorageMachine(&cfg.Connection.ObjectStorage))
}
