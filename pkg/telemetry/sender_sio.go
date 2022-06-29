package telemetry

import (
	"net/http"
	"os"

	"github.com/segmentio/analytics-go/v3"
)

const SegmentioEnvVariableName = "TESTKUBE_SEGMENTIO_KEY"

var SegmentioKey = ""

func SegmentioSender(client *http.Client, payload Payload) (out string, err error) {

	// load key from build ldflags for cli or from env for api
	if SegmentioKey == "" {
		SegmentioKey = os.Getenv(SegmentioEnvVariableName)
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
