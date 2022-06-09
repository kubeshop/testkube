package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	testkubeAnalyticsUrl = "https://analytics.testkube.io"
)

func TestkubeAnalyticsSender(client *http.Client, payload Payload) (out string, err error) {

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}

	request, err := http.NewRequest("POST", testkubeAnalyticsUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return out, err
	}

	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode > 300 {
		return out, fmt.Errorf("could not POST, statusCode: %d", resp.StatusCode)
	}
	return fmt.Sprintf("status: %d - %s", resp.StatusCode, b), err

}
