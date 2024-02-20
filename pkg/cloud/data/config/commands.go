package config

import "github.com/kubeshop/testkube/pkg/cloud/data/executor"

const (
	CmdConfigGetUniqueClusterId  executor.Command = "get_unique_cluster_id"
	CmdConfigGetTelemetryEnabled executor.Command = "get_telemetry_enabled"
	CmdConfigGet                 executor.Command = "get"
	CmdConfigUpsert              executor.Command = "upsert"
	CmdConfigGetOrganizationPlan executor.Command = "get_org_plan"
)
