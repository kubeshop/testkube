package envs

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

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
	SkipVerify                bool   `envconfig:"RUNNER_SKIP_VERIFY" default:"false"` // RUNNER_SKIP_VERIFY
	CertFile                  string `envconfig:"RUNNER_CERT_FILE"`                   // RUNNER_CERT_FILE
	KeyFile                   string `envconfig:"RUNNER_KEY_FILE"`                    // RUNNER_KEY_FILE
	CAFile                    string `envconfig:"RUNNER_CA_FILE"`
	ScrapperEnabled           bool   // RUNNER_SCRAPPERENABLED
	DataDir                   string // RUNNER_DATADIR
	GitUsername               string // RUNNER_GITUSERNAME
	GitToken                  string // RUNNER_GITTOKEN
	CompressArtifacts         bool   // RUNNER_COMPRESSARTIFACTS
	WorkingDir                string // RUNNER_WORKINGDIR
	ExecutionID               string // RUNNER_EXECUTIONID
	TestName                  string // RUNNER_TESTNAME
	ExecutionNumber           int32  // RUNNER_EXECUTIONNUMBER
	ContextType               string // RUNNER_CONTEXTTYPE
	ContextData               string // RUNNER_CONTEXTDATA
	APIURI                    string // RUNNER_APIURI
	ClusterID                 string `envconfig:"RUNNER_CLUSTERID"`                             // RUNNER_CLUSTERID
	CDEventsTarget            string `envconfig:"RUNNER_CDEVENTS_TARGET"`                       // RUNNER_CDEVENTS_TARGET
	DashboardURI              string `envconfig:"RUNNER_DASHBOARD_URI"`                         // RUNNER_DASHBOARD_URI
	CloudMode                 bool   `envconfig:"RUNNER_CLOUD_MODE"`                            // RUNNER_CLOUD_MODE
	CloudAPIKey               string `envconfig:"RUNNER_CLOUD_API_KEY"`                         // RUNNER_CLOUD_API_KEY
	CloudAPITLSInsecure       bool   `envconfig:"RUNNER_CLOUD_API_TLS_INSECURE"`                // RUNNER_CLOUD_API_TLS_INSECURE
	CloudAPIURL               string `envconfig:"RUNNER_CLOUD_API_URL"`                         // RUNNER_CLOUD_API_URL
	CloudConnectionTimeoutSec int    `envconfig:"RUNNER_CLOUD_CONNECTION_TIMEOUT" default:"10"` // RUNNER_CLOUD_CONNECTION_TIMEOUT
	CloudAPISkipVerify        bool   `envconfig:"RUNNER_CLOUD_API_SKIP_VERIFY" default:"false"` // RUNNER_CLOUD_API_SKIP_VERIFY
	ProMode                   bool   `envconfig:"RUNNER_PRO_MODE"`                              // RUNNER_PRO_MODE
	ProAPIKey                 string `envconfig:"RUNNER_PRO_API_KEY"`                           // RUNNER_PRO_API_KEY
	ProAPITLSInsecure         bool   `envconfig:"RUNNER_PRO_API_TLS_INSECURE"`                  // RUNNER_PRO_API_TLS_INSECURE
	ProAPIURL                 string `envconfig:"RUNNER_PRO_API_URL"`                           // RUNNER_PRO_API_URL
	ProConnectionTimeoutSec   int    `envconfig:"RUNNER_PRO_CONNECTION_TIMEOUT" default:"10"`   // RUNNER_PRO_CONNECTION_TIMEOUT
	ProAPISkipVerify          bool   `envconfig:"RUNNER_PRO_API_SKIP_VERIFY" default:"false"`   // RUNNER_PRO_API_SKIP_VERIFY
	ProAPICertFile            string `envconfig:"RUNNER_PRO_API_CERT_FILE"`                     // RUNNER_PRO_API_CERT_FILE
	ProAPIKeyFile             string `envconfig:"RUNNER_PRO_API_KEY_FILE"`                      // RUNNER_PRO_API_KEY_FILE
	ProAPICAFile              string `envconfig:"RUNNER_PRO_API_CA_FILE"`                       // RUNNER_PRO_API_CA_FILE
	SlavesConfigs             string `envconfig:"RUNNER_SLAVES_CONFIGS"`                        // RUNNER_SLAVES_CONFIGS
}

// LoadTestkubeVariables loads the parameters provided as environment variables in the Test CRD
func LoadTestkubeVariables() (Params, error) {
	var params Params
	err := envconfig.Process("runner", &params)
	if err != nil {
		return params, errors.Errorf("failed to read environment variables: %v", err)
	}
	cleanDeprecatedParams(&params)
	return params, nil
}

// PrintParams shows the read parameters in logs
func PrintParams(params Params) {
	output.PrintLogf("%s Environment variables read successfully", ui.IconCheckMark)
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
	output.PrintLogf("RUNNER_CLOUD_MODE=\"%t\" - DEPRECATED: please use RUNNER_PRO_MODE instead", params.CloudMode)
	output.PrintLogf("RUNNER_CLOUD_API_TLS_INSECURE=\"%t\" - DEPRECATED: please use RUNNER_PRO_API_TLS_INSECURE instead", params.CloudAPITLSInsecure)
	output.PrintLogf("RUNNER_CLOUD_API_URL=\"%s\" - DEPRECATED: please use RUNNER_PRO_API_URL instead", params.CloudAPIURL)
	printSensitiveDeprecatedParam("RUNNER_CLOUD_API_KEY", params.CloudAPIKey, "RUNNER_PRO_API_KEY")
	output.PrintLogf("RUNNER_CLOUD_CONNECTION_TIMEOUT=%d - DEPRECATED: please use RUNNER_PRO_CONNECTION_TIMEOUT instead", params.CloudConnectionTimeoutSec)
	output.PrintLogf("RUNNER_CLOUD_API_SKIP_VERIFY=\"%t\" - DEPRECATED: please use RUNNER_PRO_API_SKIP_VERIFY instead", params.CloudAPISkipVerify)
	output.PrintLogf("RUNNER_PRO_MODE=\"%t\"", params.ProMode)
	output.PrintLogf("RUNNER_PRO_API_TLS_INSECURE=\"%t\"", params.ProAPITLSInsecure)
	output.PrintLogf("RUNNER_PRO_API_URL=\"%s\"", params.ProAPIURL)
	printSensitiveParam("RUNNER_PRO_API_KEY", params.ProAPIKey)
	output.PrintLogf("RUNNER_PRO_CONNECTION_TIMEOUT=%d", params.ProConnectionTimeoutSec)
	output.PrintLogf("RUNNER_PRO_API_SKIP_VERIFY=\"%t\"", params.ProAPISkipVerify)

}

// printSensitiveParam shows in logs if a parameter is set or not
func printSensitiveParam(name string, value string) {
	if len(value) == 0 {
		output.PrintLogf("%s=\"\"", name)
	} else {
		output.PrintLogf("%s=\"********\"", name)
	}
}

// printSensitiveDeprecatedParam shows in logs if a parameter is set or not
func printSensitiveDeprecatedParam(name string, value string, newName string) {
	if len(value) == 0 {
		output.PrintLogf("%s=\"\" - DEPRECATED: please use %s instead", name, newName)
	} else {
		output.PrintLogf("%s=\"********\" - DEPRECATED: please use %s instead", name, newName)
	}
}

// cleanDeprecatedParams makes sure deprecated parameter values are set in replacements
func cleanDeprecatedParams(params *Params) {
	if !params.ProMode && params.CloudMode {
		params.ProMode = params.CloudMode
	}

	if params.ProAPIKey == "" && params.CloudAPIKey != "" {
		params.ProAPIKey = params.CloudAPIKey
	}

	if !params.ProAPITLSInsecure && params.CloudAPITLSInsecure {
		params.ProAPITLSInsecure = params.CloudAPITLSInsecure
	}

	if params.ProAPIURL == "" && params.CloudAPIURL != "" {
		params.ProAPIURL = params.CloudAPIURL
	}

	if params.ProConnectionTimeoutSec == 0 && params.CloudConnectionTimeoutSec != 0 {
		params.ProConnectionTimeoutSec = params.CloudConnectionTimeoutSec
	}

	if !params.ProAPISkipVerify && params.CloudAPISkipVerify {
		params.ProAPISkipVerify = params.CloudAPISkipVerify
	}
}
