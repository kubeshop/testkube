package analytics

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tools/commands"
	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

var TestkubeMeasurementID = "" //this is default but it can be set using ldflag -X github.com/kubeshop/testkube/pkg/analytics.TestkubeMeasurementID=G-B6KY2SF30K
var TestkubeApiSecret = ""

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
func SendAnonymousInfo() (string, error) {
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
						AppName:       "testkube-api-server",
					},
				}},
		}

		return sendDataToGA(payload)
	}
	return "", nil
}

// SendAnonymousCmdInfo will send CLI event to GA
func SendAnonymousCmdInfo(cmd *cobra.Command) (string, error) {

	// get all sub-commands passed to cli
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}

	payload := Payload{
		ClientID: MachineID(),
		Events: []Event{
			{
				Name: text.Slug(command),
				Params: Params{
					EventCount:    1,
					EventCategory: "execution",
					AppVersion:    commands.Version,
					AppName:       "kubectl-testkube",
				},
			}},
	}

	return sendDataToGA(payload)
}

// SendAnonymousCmdInfo will send CLI event to GA
func SendAnonymousAPIInfo(host, path string) (string, error) {
	payload := Payload{
		ClientID: MachineID(),
		Events: []Event{
			{
				Name: text.Slug(path),
				Params: Params{
					EventCount:       1,
					EventCategory:    "api-request",
					AppVersion:       api.Version,
					AppName:          "testkube-api-server",
					CustomDimensions: host,
				},
			}},
	}

	return sendDataToGA(payload)
}

func sendDataToGA(payload Payload) (out string, err error) {
	log.DefaultLogger.Debugw("sending ga payload", "payload", payload)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}

	request, err := http.NewRequest("POST", fmt.Sprintf(gaUrl, TestkubeMeasurementID, TestkubeApiSecret), bytes.NewBuffer(jsonData))
	if err != nil {
		return out, err
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
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
