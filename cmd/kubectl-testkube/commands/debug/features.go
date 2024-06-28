package debug

import "strings"

const (
	showPods                   = "pods"
	showServices               = "services"
	showIngresses              = "ingresses"
	showStorageClasses         = "storageclasses"
	showEvents                 = "events"
	showControlPlaneConnection = "connection"
	showRoundtrip              = "roundtrip"

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
	showControlPlaneConnection,
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
	showControlPlaneConnection,
	showRoundtrip,
}

var agentFeaturesStr = strings.Join(agentFeatures, ",")
var controlPlaneFeaturesStr = strings.Join(controlPlaneFeatures, ",")
