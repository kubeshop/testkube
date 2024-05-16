package telemetry

import (
	"fmt"
	"net/http"
	"os"

	"github.com/segmentio/analytics-go/v3"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

const SegmentioEnvVariableName = "TESTKUBE_SEGMENTIO_KEY"
const CloudEnvVariableName = "TESTKUBE_CLOUD_API_KEY"
const ProEnvVariableName = "TESTKUBE_PRO_API_KEY"

// Brew builds can't be parametrized so we are embedding this one
var SegmentioKey = "jELokNFNcLeQhxdpGF47PcxCtOLpwVuu"
var CloudSegmentioKey = ""

const AppBuild string = "oss"

func StdLogger() analytics.Logger {
	return stdLogger{}
}

type stdLogger struct {
}

func (l stdLogger) Logf(format string, args ...interface{}) {
	log.DefaultLogger.Debugw("sending telemetry data", "info", fmt.Sprintf(format, args...))
}

func (l stdLogger) Errorf(format string, args ...interface{}) {
	log.DefaultLogger.Debugw("sending telemetry data", "error", fmt.Sprintf(format, args...))
}

// SegmentioSender sends ananymous telemetry data to segment.io
// TODO refactor Sender func as out is not needed (use debug loggers to log output)
func SegmentioSender(client *http.Client, payload Payload) (out string, err error) {

	// TODO consider removing this as CLI has fixed key and API overrides it in build time
	if key, ok := os.LookupEnv(SegmentioEnvVariableName); ok {
		SegmentioKey = key
	}
	key := utils.GetEnvVarWithDeprecation(ProEnvVariableName, CloudEnvVariableName, "")
	if key != "" {
		SegmentioKey = CloudSegmentioKey
	}

	segmentio, err := analytics.NewWithConfig(SegmentioKey, analytics.Config{Logger: StdLogger()})
	if err != nil {
		return out, err
	}
	defer segmentio.Close()

	for _, event := range payload.Events {
		err := segmentio.Enqueue(mapEvent(payload.UserID, event))
		if err != nil {
			return out, err
		}
	}

	return
}

// TODO refactor Event model to be more generic not GA4 like after removoing debug and GA4
func mapEvent(userID string, event Event) analytics.Track {
	return analytics.Track{
		Event:      event.Name,
		UserId:     userID,
		Properties: mapProperties(event.Params),
		Context: &analytics.Context{
			App: analytics.AppInfo{
				Name:    event.Params.AppName,
				Version: event.Params.AppVersion,
				Build:   AppBuild,
			},
		},
	}
}

func mapProperties(params Params) analytics.Properties {
	properties := analytics.NewProperties().
		Set("name", params.AppName).
		Set("version", params.AppVersion).
		Set("arch", params.Architecture).
		Set("os", params.OperatingSystem).
		Set("clusterId", params.ClusterID).
		Set("eventCategory", params.EventCategory).
		Set("host", params.Host).
		Set("contextType", params.Context.Type).
		Set("cloudOrganizationId", params.Context.OrganizationId).
		Set("cloudEnvironmentId", params.Context.EnvironmentId).
		Set("machineId", params.MachineID).
		Set("clusterType", params.ClusterType).
		Set("errorType", params.ErrorType).
		Set("errorStackTrace", params.ErrorStackTrace)

	if params.DataSource != "" {
		properties = properties.Set("dataSource", params.DataSource)
	}

	if params.TestType != "" {
		properties = properties.Set("testType", params.TestType)
	}

	if params.DurationMs != 0 {
		properties = properties.Set("durationMs", params.DurationMs)
	}

	if params.Status != "" {
		properties = properties.Set("status", params.Status)
	}

	if params.TestSource != "" {
		properties = properties.Set("testSource", params.TestSource)
	}

	if params.TestSuiteSteps != 0 {
		properties = properties.Set("testSuiteSteps", params.TestSuiteSteps)
	}

	return properties
}
