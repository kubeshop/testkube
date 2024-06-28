package debug

import "strings"

const (
	showPods              = "pods"
	showServices          = "services"
	showIngresses         = "ingresses"
	showStorageClasses    = "storageclasses"
	showEvents            = "events"
	showCLIToControlPlane = "connection"
	showRoundtrip         = "roundtrip"

	showApiLogs    = "api"
	showWorkerLogs = "worker"
	showUiLogs     = "ui"
	showDexLogs    = "dex"
	showNatsLogs   = "nats"
	showMongoLogs  = "mongo"
	showMinioLogs  = "minio"
)

var controlPlaneFeatures = []string{
	showPods,
	showServices,
	showIngresses,
	showStorageClasses,
	showEvents,
	showNatsLogs,
	showCLIToControlPlane,
	showApiLogs,
	showNatsLogs,
	showMongoLogs,
	showDexLogs,
	showUiLogs,
	showWorkerLogs,
}

var agentFeatures = []string{
	showPods,
	showServices,
	showIngresses,
	showEvents,
	showNatsLogs,
	showCLIToControlPlane,
	showRoundtrip,
}

var agentFeaturesStr = strings.Join(agentFeatures, ",")
var controlPlaneFeaturesStr = strings.Join(controlPlaneFeatures, ",")
