package envs

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Params are the environment variables provided by the Testkube api-server
type Params struct {
	StorageEndpoint           string `envconfig:"RUNNER_STORAGE_ENDPOINT"`
	StorageAccessKeyID        string `envconfig:"RUNNER_STORAGE_ACCESSKEYID"`
	StorageSecretAccessKey    string `envconfig:"RUNNER_STORAGE_SECRETACCESSKEY"`
	StorageRegion             string `envconfig:"RUNNER_STORAGE_REGION"`
	StorageToken              string `envconfig:"RUNNER_STORAGE_TOKEN"`
	StorageBucket             string `envconfig:"RUNNER_STORAGE_BUCKET"`
	StorageSSL                bool   `envconfig:"RUNNER_STORAGE_SSL" default:"false"`
	StorageSkipVerify         bool   `envconfig:"RUNNER_STORAGE_SKIP_VERIFY" default:"false"`
	StorageCertFile           string `envconfig:"RUNNER_STORAGE_CERT_FILE"`
	StorageKeyFile            string `envconfig:"RUNNER_STORAGE_KEY_FILE"`
	StorageCAFile             string `envconfig:"RUNNER_STORAGE_CA_FILE"`
	ScrapperEnabled           bool   `envconfig:"RUNNER_SCRAPPERENABLED" default:"false"`
	DataDir                   string `envconfig:"RUNNER_DATADIR"`
	GitUsername               string `envconfig:"RUNNER_GITUSERNAME"`
	GitToken                  string `envconfig:"RUNNER_GITTOKEN"`
	CompressArtifacts         bool   `envconfig:"RUNNER_COMPRESSARTIFACTS" default:"false"`
	WorkingDir                string `envconfig:"RUNNER_WORKINGDIR"`
	ExecutionID               string `envconfig:"RUNNER_EXECUTIONID"`
	TestName                  string `envconfig:"RUNNER_TESTNAME"`
	ExecutionNumber           int32  `envconfig:"RUNNER_EXECUTIONNUMBER"`
	ContextType               string `envconfig:"RUNNER_CONTEXTTYPE"`
	ContextData               string `envconfig:"RUNNER_CONTEXTDATA"`
	APIURI                    string `envconfig:"RUNNER_APIURI"`
	ClusterID                 string `envconfig:"RUNNER_CLUSTERID"`
	CDEventsTarget            string `envconfig:"RUNNER_CDEVENTS_TARGET"`
	DashboardURI              string `envconfig:"RUNNER_DASHBOARD_URI"`
	CloudMode                 bool   `envconfig:"RUNNER_CLOUD_MODE" default:"false"`
	CloudAPIKey               string `envconfig:"RUNNER_CLOUD_API_KEY"`
	CloudAPIURL               string `envconfig:"RUNNER_CLOUD_API_URL"`
	AgentConnectionTimeoutSec int    `envconfig:"RUNNER_CLOUD_CONNECTION_TIMEOUT" default:"10"`
	AgentInsecure             bool   `envconfig:"RUNNER_AGENT_INSECURE" default:"false"`
	AgentSkipVerify           bool   `envconfig:"RUNNER_AGENT_SKIP_VERIFY" default:"false"`
	AgentCertFile             string `envconfig:"RUNNER_AGENT_CERT_FILE"`
	AgentKeyFile              string `envconfig:"RUNNER_AGENT_KEY_FILE"`
	AgentCAFile               string `envconfig:"RUNNER_AGENT_CA_FILE"`
	SlavesConfigs             string `envconfig:"RUNNER_SLAVES_CONFIGS"`
}

// LoadTestkubeVariables loads the parameters provided as environment variables in the Test CRD
func LoadTestkubeVariables() (Params, error) {
	var params Params
	if err := envconfig.Process("runner", &params); err != nil {
		return params, errors.Errorf("failed to read environment variables: %v", err)
	}

	return params, nil
}

// PrintParams shows the read parameters in logs
func PrintParams(params Params) {
	output.PrintLogf("%s Environment variables read successfully", ui.IconCheckMark)
	output.PrintLogf("RUNNER_STORAGE_ENDPOINT=\"%s\"", params.StorageEndpoint)
	printSensitiveParam("RUNNER_STORAGE_ACCESSKEYID", params.StorageAccessKeyID)
	printSensitiveParam("RUNNER_STORAGE_SECRETACCESSKEY", params.StorageSecretAccessKey)
	output.PrintLogf("RUNNER_STORAGE_REGION=\"%s\"", params.StorageRegion)
	printSensitiveParam("RUNNER_STORAGE_TOKEN", params.StorageToken)
	output.PrintLogf("RUNNER_STORAGE_BUCKET=\"%s\"", params.StorageBucket)
	output.PrintLogf("RUNNER_STORAGE_SSL=%t", params.StorageSSL)
	output.PrintLogf("RUNNER_STORAGE_SKIP_VERIFY=%t", params.StorageSkipVerify)
	output.PrintLogf("RUNNER_STORAGE_CERT_FILE=\"%s\"", params.StorageCertFile)
	output.PrintLogf("RUNNER_STORAGE_KEY_FILE=\"%s\"", params.StorageKeyFile)
	output.PrintLogf("RUNNER_STORAGE_CA_FILE=\"%s\"", params.StorageCAFile)
	output.PrintLogf("RUNNER_SCRAPPERENABLED=\"%t\"", params.ScrapperEnabled)
	output.PrintLogf("RUNNER_GITUSERNAME=\"%s\"", params.GitUsername)
	printSensitiveParam("RUNNER_GITTOKEN", params.GitToken)
	output.PrintLogf("RUNNER_DATADIR=\"%s\"", params.DataDir)
	output.PrintLogf("RUNNER_COMPRESSARTIFACTS=\"%t\"", params.CompressArtifacts)
	output.PrintLogf("RUNNER_WORKINGDIR=\"%s\"", params.WorkingDir)
	output.PrintLogf("RUNNER_EXECUTIONID=\"%s\"", params.ExecutionID)
	output.PrintLogf("RUNNER_TESTNAME=\"%s\"", params.TestName)
	output.PrintLogf("RUNNER_EXECUTIONNUMBER=\"%d\"", params.ExecutionNumber)
	output.PrintLogf("RUNNER_CONTEXTTYPE=\"%s\"", params.ContextType)
	output.PrintLogf("RUNNER_CONTEXTDATA=\"%s\"", params.ContextData)
	output.PrintLogf("RUNNER_APIURI=\"%s\"", params.APIURI)
	output.PrintLogf("RUNNER_CLUSTERID=\"%s\"", params.ClusterID)
	output.PrintLogf("RUNNER_CDEVENTS_TARGET=\"%s\"", params.CDEventsTarget)
	output.PrintLogf("RUNNER_DASHBOARD_URI=\"%s\"", params.DashboardURI)
	output.PrintLogf("RUNNER_CLOUD_MODE=\"%t\"", params.CloudMode)
	output.PrintLogf("RUNNER_CLOUD_API_URL=\"%s\"", params.CloudAPIURL)
	printSensitiveParam("RUNNER_CLOUD_API_KEY", params.CloudAPIKey)
	output.PrintLogf("RUNNER_AGENT_INSECURE=\"%t\"", params.AgentInsecure)
	output.PrintLogf("RUNNER_AGENT_SKIP_VERIFY=\"%t\"", params.AgentSkipVerify)
	output.PrintLogf("RUNNER_AGENT_CERT_FILE=\"%s\"", params.AgentCertFile)
	output.PrintLogf("RUNNER_AGENT_KEY_FILE=\"%s\"", params.AgentKeyFile)
	output.PrintLogf("RUNNER_AGENT_CA_FILE=\"%s\"", params.AgentCAFile)
	output.PrintLogf("RUNNER_AGENT_CONNECTION_TIMEOUT=%d", params.AgentConnectionTimeoutSec)
}

// printSensitiveParam shows in logs if a parameter is set or not
func printSensitiveParam(name string, value string) {
	if len(value) == 0 {
		output.PrintLogf("%s=\"\"", name)
	} else {
		output.PrintLogf("%s=\"********\"", name)
	}
}
