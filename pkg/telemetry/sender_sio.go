package telemetry

import (
	"net/http"
	"os"

	"github.com/segmentio/analytics-go/v3"
)

const SegmentioEnvVariableName = "TESTKUBE_SEGMENTIO_KEY"

// Brew builds can't be parametrized so we are embedding this one
var SegmentioKey = "jELokNFNcLeQhxdpGF47PcxCtOLpwVuu"

// SegmentioSender sends ananymous telemetry data to segment.io
// TODO refactor Sender func as out is not needed (use debug loggers to log output)
func SegmentioSender(client *http.Client, payload Payload) (out string, err error) {

	// TODO consider removing this as CLI has fixed key and API overrides it in build time
	if key, ok := os.LookupEnv(SegmentioEnvVariableName); ok {
		SegmentioKey = key
	}

	segmentio := analytics.New(SegmentioKey)
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
		Set("machineId", params.MachineID)

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
