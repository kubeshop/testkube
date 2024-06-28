package debug

import "strings"

const (
	showPods           = "pods"
	showServices       = "services"
	showIngresses      = "ingresses"
	showStorageClasses = "storageclasses"
	showEvents         = "events"
	showDebug          = "debug"
	showConnection     = "connection"

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
	showDebug,
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
	showDebug,
	showConnection,
}

var agentFeaturesStr = strings.Join(agentFeatures, ",")
var controlPlaneFeaturesStr = strings.Join(controlPlaneFeatures, ",")
