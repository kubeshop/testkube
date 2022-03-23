package analytics

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/go-resty/resty/v2"

	"github.com/kubeshop/testkube/cmd/tools/commands"
)

var testkubeMeasurementID = "" //this is default but it can be set using ldflag -X github.com/kubeshop/testkube/pkg/analytics.testkubeMeasurementID=G-B6KY2SF30K
var testkubeApiSecret = ""

const gaUrl = "https://www.google-analytics.com/mp/collect?measurement_id=%s&api_secret=%s"

type Params struct {
	EventCount       int64  `json:"event_count,omitempty"`
	EventCategory    string `json:"even_category,omitempty"`
	AppVersion       string `json:"app_version,omitempty"`
	AppName          string `json:"app_name,omitempty"`
	CustomDimensions string `json:"custom_dimensions,omitempty"`
	DataSource       string `json:"data_source,omitempty"`
}
type Event struct {
	Name   string `json:"name"`
	Params Params `json:"params,omitempty"`
}
type Payload struct {
	ClientID string  `json:"client_id"`
	Events   []Event `json:"events"`
}

// SendAnonymousInfo will send event to GA
func SendAnonymousInfo() {

	var isEnabled bool
	if val, ok := os.LookupEnv("TESTKUBE_ANALYTICS_ENABLED"); ok {
		isEnabled, _ = strconv.ParseBool(val)
	}
	if isEnabled {
		payload := Payload{
			ClientID: MachineID(),
			Events: []Event{
				Event{
					Name: "testkube-heartbeat",

					Params: Params{
						EventCount:    1,
						EventCategory: "beacon",
						AppVersion:    commands.Version,
						AppName:       "testkube",
						DataSource:    "api-server",
					},
				}},
		}

		sendDataToGA(payload)
	}
}

// SendAnonymouscmdInfo will send CLI event to GA
func SendAnonymouscmdInfo() {
	event := "command"
	command := []string{}
	if len(os.Args) > 1 {
		command = os.Args[1:]
		event = command[0]
	}

	payload := Payload{
		ClientID: MachineID(),
		Events: []Event{
			Event{
				Name: event,
				Params: Params{
					EventCount:       1,
					EventCategory:    "execution",
					AppVersion:       commands.Version,
					AppName:          "testkube",
					CustomDimensions: strings.Join(command, " "),
				},
			}},
	}

	sendDataToGA(payload)

}

func sendDataToGA(data Payload) error {

	jsonbyte, err := json.Marshal(data)
	if err != nil {
		return err
	}
	json := string(jsonbyte)
	fmt.Println(json)

	restyClient := resty.New()
	resp, err := restyClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Post(fmt.Sprintf(gaUrl, testkubeMeasurementID, testkubeApiSecret))
	if err != nil {
		return err
	}

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("Could not POST, statusCode: %d", resp.StatusCode())
	}
	return nil
}

// MachineID returns unique user machine ID
func MachineID() string {
	id, _ := generate()
	return id
}

// Generate returns protected id for the current machine
func generate() (string, error) {
	id, err := machineid.ProtectedID("testkube")
	if err != nil {
		return fromHostname()
	}
	return id, err
}

// fromHostname generates a machine id hash from hostname
func fromHostname() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}
	sum := md5.Sum([]byte(name))
	return hex.EncodeToString(sum[:]), nil
}
