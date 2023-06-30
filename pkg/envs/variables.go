package envs

import (
	"github.com/pkg/errors"

	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Params are the environment variables provided by the Testkube api-server
type Params struct {
	Endpoint                  string // RUNNER_ENDPOINT
	AccessKeyID               string // RUNNER_ACCESSKEYID
	SecretAccessKey           string // RUNNER_SECRETACCESSKEY
	Region                    string // RUNNER_REGION
	Token                     string // RUNNER_TOKEN
	Bucket                    string // RUNNER_BUCKET
	Ssl                       bool   // RUNNER_SSL
	ScrapperEnabled           bool   // RUNNER_SCRAPPERENABLED
	DataDir                   string // RUNNER_DATADIR
	GitUsername               string // RUNNER_GITUSERNAME
	GitToken                  string // RUNNER_GITTOKEN
	ClusterID                 string `envconfig:"RUNNER_CLUSTERID"`                             // RUNNER_CLUSTERID
	CDEventsTarget            string `envconfig:"RUNNER_CDEVENTS_TARGET"`                       // RUNNER_CDEVENTS_TARGET
	DashboardURI              string `envconfig:"RUNNER_DASHBOARD_URI"`                         // RUNNER_DASHBOARD_URI
	CloudMode                 bool   `envconfig:"RUNNER_CLOUD_MODE"`                            // RUNNER_CLOUD_MODE
	CloudAPIKey               string `envconfig:"RUNNER_CLOUD_API_KEY"`                         // RUNNER_CLOUD_API_KEY
	CloudAPITLSInsecure       bool   `envconfig:"RUNNER_CLOUD_API_TLS_INSECURE"`                // RUNNER_CLOUD_API_TLS_INSECURE
	CloudAPIURL               string `envconfig:"RUNNER_CLOUD_API_URL"`                         // RUNNER_CLOUD_API_URL
	CloudConnectionTimeoutSec int    `envconfig:"RUNNER_CLOUD_CONNECTION_TIMEOUT" default:"10"` // RUNNER_CLOUD_CONNECTION_TIMEOUT
}

// LoadTestkubeVariables loads the parameters provided as environment variables in the Test CRD
func LoadTestkubeVariables() (Params, error) {
	var params Params
	output.PrintLogf("%s Reading environment variables...", ui.IconWorld)
	err := envconfig.Process("runner", &params)
	if err != nil {
		output.PrintLogf("%s Failed to read environment variables: %s", ui.IconCross, err.Error())
		return params, errors.Errorf("failed to read environment variables: %v", err)
	}
	output.PrintLogf("%s Environment variables read successfully", ui.IconCheckMark)
	printParams(params)

	return params, nil
}

// printParams shows the read parameters in logs
func printParams(params Params) {
	output.PrintLogf("RUNNER_ENDPOINT=\"%s\"", params.Endpoint)
	printSensitiveParam("RUNNER_ACCESSKEYID", params.AccessKeyID)
	printSensitiveParam("RUNNER_SECRETACCESSKEY", params.SecretAccessKey)
	output.PrintLogf("RUNNER_REGION=\"%s\"", params.Region)
	printSensitiveParam("RUNNER_TOKEN", params.Token)
	output.PrintLogf("RUNNER_BUCKET=\"%s\"", params.Bucket)
	output.PrintLogf("RUNNER_SSL=%t", params.Ssl)
	output.PrintLogf("RUNNER_SCRAPPERENABLED=\"%t\"", params.ScrapperEnabled)
	output.PrintLogf("RUNNER_GITUSERNAME=\"%s\"", params.GitUsername)
	printSensitiveParam("RUNNER_GITTOKEN", params.GitToken)
	output.PrintLogf("RUNNER_DATADIR=\"%s\"", params.DataDir)
	output.PrintLogf("RUNNER_CLUSTERID=\"%s\"", params.ClusterID)
	output.PrintLogf("RUNNER_CDEVENTS_TARGET=\"%s\"", params.CDEventsTarget)
	output.PrintLogf("RUNNER_DASHBOARD_URI=\"%s\"", params.DashboardURI)
	output.PrintLogf("RUNNER_CLOUD_MODE=\"%t\"", params.CloudMode)
	output.PrintLogf("RUNNER_CLOUD_API_TLS_INSECURE=\"%t\"", params.CloudAPITLSInsecure)
	output.PrintLogf("RUNNER_CLOUD_API_URL=\"%s\"", params.CloudAPIURL)
	printSensitiveParam("RUNNER_CLOUD_API_KEY", params.CloudAPIKey)
	output.PrintLogf("RUNNER_CLOUD_CONNECTION_TIMEOUT=%d", params.CloudConnectionTimeoutSec)
}

// printSensitiveParam shows in logs if a parameter is set or not
func printSensitiveParam(name string, value string) {
	if len(value) == 0 {
		output.PrintLogf("%s=\"\"", name)
	} else {
		output.PrintLogf("%s=\"********\"", name)
	}
}
