package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/log"
)

const (
	gaUrl           = "https://www.google-analytics.com/mp/collect?measurement_id=%s&api_secret=%s"
	gaValidationUrl = "https://www.google-analytics.com/debug/mp/collect?measurement_id=%s&api_secret=%s"
)

var (
	TestkubeMeasurementID     = "" //this is default but it can be set using ldflag -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=G-B6KY2SF30K
	TestkubeMeasurementSecret = ""
)

func GoogleAnalyticsSender(client *http.Client, payload Payload) (out string, err error) {
	out, err = sendValidationRequest(payload)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}

	request, err := http.NewRequest("POST", fmt.Sprintf(gaUrl, TestkubeMeasurementID, TestkubeMeasurementSecret), bytes.NewBuffer(jsonData))
	if err != nil {
		return out, err
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)

	if resp.StatusCode > 300 {
		return out, fmt.Errorf("could not POST, statusCode: %d", resp.StatusCode)
	}
	return fmt.Sprintf("status: %d - %s", resp.StatusCode, b), err
}

func sendValidationRequest(payload Payload) (out string, err error) {
	log.DefaultLogger.Debugw("sending ga payload to validate", "payload", payload)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}

	uri := fmt.Sprintf(gaValidationUrl, TestkubeMeasurementID, TestkubeMeasurementSecret)

	request, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return out, err
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)

	if resp.StatusCode > 300 {
		return out, fmt.Errorf("could not POST, statusCode: %d", resp.StatusCode)
	}
	return fmt.Sprintf("status: %d - %s", resp.StatusCode, b), err
}
