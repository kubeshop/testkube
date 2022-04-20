package analytics

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tools/commands"
	"github.com/kubeshop/testkube/internal/pkg/api"
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
				{
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

// SendAnonymousCmdInfo will send CLI event to GA
func SendAnonymousCmdInfo(cmd *cobra.Command) error {

	// get all sub-commands passed to cli
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}

	args := []string{}
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	payload := Payload{
		ClientID: MachineID(),
		Events: []Event{
			{
				Name: command,
				Params: Params{
					EventCount:       1,
					EventCategory:    "execution",
					AppVersion:       commands.Version,
					AppName:          "testkube",
					CustomDimensions: strings.Join(args, " "),
					DataSource:       "kubectl-testkube",
				},
			}},
	}

	return sendDataToGA(payload)
}

// SendAnonymousCmdInfo will send CLI event to GA
func SendAnonymousAPIInfo(path string) error {
	payload := Payload{
		ClientID: MachineID(),
		Events: []Event{
			{
				Name: path,
				Params: Params{
					EventCount:    1,
					EventCategory: "api-request",
					AppVersion:    api.Version,
					AppName:       "testkube",
					DataSource:    "api-server",
				},
			}},
	}

	return sendDataToGA(payload)
}

func sendDataToGA(data Payload) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", fmt.Sprintf(gaUrl, testkubeMeasurementID, testkubeApiSecret), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		return fmt.Errorf("could not POST, statusCode: %d", resp.StatusCode)
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
