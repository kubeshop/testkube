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
	"runtime"
	"strconv"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tools/commands"
	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

var TestkubeMeasurementID = "" //this is default but it can be set using ldflag -X github.com/kubeshop/testkube/pkg/analytics.TestkubeMeasurementID=G-B6KY2SF30K
var TestkubeMeasurementSecret = ""

const gaUrl = "https://www.google-analytics.com/mp/collect?measurement_id=%s&api_secret=%s"
const gaValidationUrl = "https://www.google-analytics.com/debug/mp/collect?measurement_id=%s&api_secret=%s"

type Params struct {
	EventCount       int64  `json:"event_count,omitempty"`
	EventCategory    string `json:"event_category,omitempty"`
	AppVersion       string `json:"app_version,omitempty"`
	AppName          string `json:"app_name,omitempty"`
	CustomDimensions string `json:"custom_dimensions,omitempty"`
	DataSource       string `json:"data_source,omitempty"`
	Host             string `json:"host,omitempty"`
	MachineID        string `json:"machine_id,omitempty"`
	ClusterID        string `json:"cluster_id,omitempty"`
	OperatingSystem  string `json:"operating_system,omitempty"`
	Architecture     string `json:"architecture,omitempty"`
}
type Event struct {
	Name   string `json:"name"`
	Params Params `json:"params,omitempty"`
}
type Payload struct {
	UserID   string  `json:"user_id,omitempty"`
	ClientID string  `json:"client_id,omitempty"`
	Events   []Event `json:"events,omitempty"`
}

// SendServerStartAnonymousInfo will send event to GA
func SendServerStartAnonymousInfo() (string, error) {
	var isEnabled bool
	if val, ok := os.LookupEnv("TESTKUBE_ANALYTICS_ENABLED"); ok {
		isEnabled, _ = strconv.ParseBool(val)
	}
	if isEnabled {
		machineID := MachineID()
		payload := Payload{
			ClientID: machineID,
			UserID:   machineID,
			Events: []Event{
				{
					Name: "testkube_heartbeat",
					Params: Params{
						EventCount:      1,
						EventCategory:   "beacon",
						AppVersion:      commands.Version,
						AppName:         "testkube-api-server",
						MachineID:       machineID,
						OperatingSystem: runtime.GOOS,
						Architecture:    runtime.GOARCH,
					},
				}},
		}

		return sendDataToGA(payload)
	}
	return "", nil
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
		ClientID: machineID,
		UserID:   machineID,
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

// SendCmdInit will send CLI event to GA
func SendCmdInit(cmd *cobra.Command, version string) (string, error) {
	machineID := MachineID()

	payload := Payload{
		ClientID: machineID,
		UserID:   machineID,
		Events: []Event{
			{
				Name: "init",
				Params: Params{
					EventCount:      1,
					EventCategory:   "execution",
					AppVersion:      version,
					AppName:         "kubectl-testkube",
					MachineID:       machineID,
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
				},
			}},
	}

	out, err := sendValidationRequest(payload)
	ui.Debug("init event validation output", out)
	if err != nil {
		ui.Debug("init event validation error", err.Error())
	}

	return sendDataToGA(payload)
}

// SendAnonymousCmdInfo will send CLI event to GA
func SendAnonymousAPIRequestInfo(host, path, version, method, clusterId string) (string, error) {
	payload := Payload{
		ClientID: clusterId,
		UserID:   clusterId,
		Events: []Event{
			{
				Name: text.GAEventName(method + "_" + path),
				Params: Params{
					EventCount:      1,
					EventCategory:   "api-request",
					AppVersion:      version,
					AppName:         "testkube-api-server",
					Host:            AnonymizeHost(host),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					MachineID:       MachineID(),
					ClusterID:       clusterId,
				},
			}},
	}

	out, err := sendValidationRequest(payload)
	log.DefaultLogger.Debugw("validation output", "payload", payload, "out", out, "error", err)

	return sendDataToGA(payload)
}

const (
	APIHostLocal            = "local"
	APIHostExternal         = "external"
	APIHostTestkubeInternal = "testkube-internal"
)

func AnonymizeHost(host string) string {
	if strings.Contains(host, "testkube.io") {
		return APIHostTestkubeInternal
	} else if strings.Contains(host, "localhost:8088") {
		return APIHostLocal
	}

	return APIHostExternal
}

func sendDataToGA(payload Payload) (out string, err error) {
	log.DefaultLogger.Debugw("sending ga payload", "payload", payload)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}

	request, err := http.NewRequest("POST", fmt.Sprintf(gaUrl, TestkubeMeasurementID, TestkubeMeasurementSecret), bytes.NewBuffer(jsonData))
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
		return fmt.Errorf("could not POST, statusCode: %d", resp.StatusCode)
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
