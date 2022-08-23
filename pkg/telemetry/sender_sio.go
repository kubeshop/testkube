package telemetry

import (
	"net/http"
	"os"

	"github.com/segmentio/analytics-go/v3"
)

const SegmentioEnvVariableName = "TESTKUBE_SEGMENTIO_KEY"

// Brew builds can't be parametrized so we are embedding this one
var SegmentioKey = "iL0p6r5C9i35F7tRxnB0k3gB2nGh7VTK"

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
	return analytics.NewProperties().
		Set("name", params.AppName).
		Set("version", params.AppVersion).
		Set("arch", params.Architecture).
		Set("os", params.OperatingSystem).
		Set("clusterId", params.ClusterID).
		Set("eventCategory", params.EventCategory).
		Set("host", params.Host).
		Set("machineId", params.MachineID)
}
