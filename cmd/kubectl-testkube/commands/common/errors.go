package common

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

type ErrorCode string

const (
	// TKERR-1xx errors are to issues when running testkube CLI commands.

	// TKERR-11xx errors are related to missing dependencies.

	// TKErrMissingDependencyHelm is returned when kubectl is not found in $PATH.
	TKErrMissingDependencyHelm ErrorCode = "TKERR-1101"
	// TKErrMissingDependencyKubectl is returned when kubectl is not found in $PATH.
	TKErrMissingDependencyKubectl ErrorCode = "TKERR-1102"
	// TKErrMissingDependencyDatabase is returned when database can't be detected.
	TKErrMissingDependencyDatabase ErrorCode = "TKERR-1103"

	// TKERR-12xx errors are related to configuration issues.

	// TKErrConfigInitFailed is returned when configuration init fails.
	TKErrConfigInitFailed ErrorCode = "TKERR-1201"
	// TKErrInvalidInstallConfig is returned when invalid configuration is supplied when installing or upgrading.
	TKErrInvalidInstallConfig ErrorCode = "TKERR-1202"
	// TKErrInvalidDockerConfig is returned when docker client configuration is invalid.
	TKErrInvalidDockerConfig ErrorCode = "TKERR-1203"
	// TKErrInvalidRuntimeParameter is returned when invalid runtime parameters are provided.
	TKErrInvalidRuntimeParameter ErrorCode = "TKERR-1204"
	// TKErrConfigSaveFailed is returned when writing the testkube config file back to disk fails.
	TKErrConfigSaveFailed ErrorCode = "TKERR-1205"

	// TKERR-13xx errors are related to install operations.

	// TKErrHelmCommandFailed is returned when a helm command fails.
	TKErrHelmCommandFailed ErrorCode = "TKERR-1301"
	// TKErrKubectlCommandFailed is returned when a kubectl command fails.
	TKErrKubectlCommandFailed ErrorCode = "TKERR-1302"
	// TKErrDockerCommandFailed is returned when a docker command fails.
	TKErrDockerCommandFailed ErrorCode = "TKERR-1303"
	// TKErrDockerLogStreamingFailed is returned when a docker log streaming fails.
	TKErrDockerLogStreamingFailed ErrorCode = "TKERR-1304"
	// TKErrDockerLogReadingFailed is returned when a docker log reading fails.
	TKErrDockerLogReadingFailed ErrorCode = "TKERR-1305"
	// TKErrDockerInstallationFailed is returned when a docker installation fails.
	TKErrDockerInstallationFailed ErrorCode = "TKERR-1306"
	// TKErrLatestVersionFetchFailed is returned when the latest Testkube release version can't be resolved.
	TKErrLatestVersionFetchFailed ErrorCode = "TKERR-1307"
	// TKErrValuesExportFailed is returned when the installation values file can't be fetched or written out.
	TKErrValuesExportFailed ErrorCode = "TKERR-1308"

	// TKErrCleanOldMigrationJobFailed is returned in case of issues with old migration jobs.
	TKErrCleanOldMigrationJobFailed ErrorCode = "TKERR-1401"

	// TKERR-15xx errors are related to agent operations.

	// TKErrAgentGetFailed is returned when fetching an agent from the control plane fails.
	TKErrAgentGetFailed ErrorCode = "TKERR-1501"
	// TKErrAgentRotateKeyFailed is returned when rotating an agent's secret key fails.
	TKErrAgentRotateKeyFailed ErrorCode = "TKERR-1502"
	// TKErrAgentRotateRegistrationTokenFailed is returned when rotating an environment registration token fails.
	TKErrAgentRotateRegistrationTokenFailed ErrorCode = "TKERR-1503"
	// TKErrAgentWriteFailed is returned when creating, updating or deleting an agent on the control plane fails.
	// Reads use TKErrAgentGetFailed: the control plane helpers share one preamble and differ only in the final
	// call, so a read and a write fail for the same reasons and the code only has to say which was attempted.
	TKErrAgentWriteFailed ErrorCode = "TKERR-1504"

	// TKERR-16xx errors are related to marketplace operations.

	// TKErrMarketplaceFetchFailed is returned when fetching marketplace content (catalog, YAML, readme) fails.
	TKErrMarketplaceFetchFailed ErrorCode = "TKERR-1601"
	// TKErrMarketplaceWorkflowNotFound is returned when the requested workflow is not present in the catalog.
	TKErrMarketplaceWorkflowNotFound ErrorCode = "TKERR-1602"
	// TKErrMarketplaceInvalidParameter is returned for any parameter-related
	// failure: the workflow spec.config could not be parsed, a --set value
	// is malformed or references an unknown key, or the parameter values
	// could not be re-applied to the workflow YAML.
	TKErrMarketplaceInvalidParameter ErrorCode = "TKERR-1603"

	// TKERR-17xx errors are related to resource lookup operations.

	// TKErrResourceNotFound is returned when a requested resource does not exist on the API server.
	TKErrResourceNotFound ErrorCode = "TKERR-1701"

	// TKERR-18xx errors are related to authentication and Pro context setup.

	// TKErrLoginFailed is returned when the interactive user login does not complete.
	TKErrLoginFailed ErrorCode = "TKERR-1801"
	// TKErrOrgResolutionFailed is returned when the Pro organization can't be resolved.
	TKErrOrgResolutionFailed ErrorCode = "TKERR-1802"
	// TKErrEnvResolutionFailed is returned when the Pro environment can't be resolved.
	TKErrEnvResolutionFailed ErrorCode = "TKERR-1803"
	// TKErrContextSaveFailed is returned when the resolved Pro context can't be stored in the config file.
	TKErrContextSaveFailed ErrorCode = "TKERR-1804"
	// TKErrOrgEnvNamesFetchFailed is returned when the display names of the context's organization and environment can't be fetched.
	TKErrOrgEnvNamesFetchFailed ErrorCode = "TKERR-1805"
	// TKErrControlPlaneDiscoveryFailed is returned when the Control Plane can't be reached or does not answer with its public info.
	TKErrControlPlaneDiscoveryFailed ErrorCode = "TKERR-1806"
)

const helpUrl = "https://testkubeworkspace.slack.com"

// ConfigFileHint is the recovery hint for any failure to read or write
// the CLI config file.
const ConfigFileHint = "Check is the Testkube config file (~/.testkube/config.json) accessible and has right permissions"

type CLIError struct {
	Code            ErrorCode
	Title           string
	Description     string
	ActualError     error
	StackTrace      string
	MoreInfo        string
	ExecutedCommand string
	Telemetry       *ErrorTelemetry
}

type ErrorTelemetry struct {
	Command *cobra.Command
	Step    string
	Type    string
	License string
}

func (e *CLIError) AddTelemetry(cmd *cobra.Command, step, errType, license string) {
	e.Telemetry = &ErrorTelemetry{
		Command: cmd,
		Step:    step,
		Type:    errType,
		License: license,
	}
}

func (e *CLIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Description)
}

func (e *CLIError) Print() {
	// Errors belong on stderr. The ui.ExitOnError and ui.Failf calls these
	// CLIErrors replace both write there, while pterm's default writer is
	// stdout - so every printer below has to be pointed at stderr explicitly,
	// or converting a call site silently moves its output between streams.
	w := os.Stderr

	pterm.DefaultHeader.WithWriter(w).Println("Testkube Error")

	pterm.DefaultSection.WithWriter(w).Println("Error Details")

	cmd := ""
	if e.ExecutedCommand != "" {
		pterm.Fprintln(w, pterm.FgDarkGray.Sprintf("Executed command: %s", e.ExecutedCommand))
		params := strings.Split(e.ExecutedCommand, " ")
		if len(params) > 0 {
			cmd = params[0]
		}
	}

	items := []pterm.BulletListItem{
		{Level: 0, Text: pterm.Sprintf("[%s]: %s", e.Code, e.Title), TextStyle: pterm.NewStyle(pterm.FgRed)},
		{Level: 0, Text: pterm.Sprintf("%s", e.Description), TextStyle: pterm.NewStyle(pterm.FgLightWhite)},
	}
	if e.MoreInfo != "" {
		items = append(items, pterm.BulletListItem{Level: 0, Text: pterm.Sprintf("%s", e.MoreInfo), TextStyle: pterm.NewStyle(pterm.FgGray)})
	}
	pterm.DefaultBulletList.WithWriter(w).WithItems(items).Render()
	if cmd != "" {
		pterm.DefaultBox.WithWriter(w).Printfln("Error description is provided in context of binary execution %s", cmd)
	}

	pterm.Fprintln(w)
	pterm.Fprintln(w, "Let us help you!")
	pterm.Fprintln(w, pterm.Sprintf("Come say hi on Slack: %s", helpUrl))
}

func NewCLIError(code ErrorCode, title, moreInfoURL string, err error) *CLIError {
	return &CLIError{
		Code:        code,
		Title:       title,
		Description: err.Error(),
		ActualError: err,
		MoreInfo:    moreInfoURL,
		StackTrace:  fmt.Sprintf("%+v", err),
	}
}

func (err *CLIError) WithExecutedCommand(executedCommand string) *CLIError {
	err.ExecutedCommand = executedCommand
	return err
}

// HandleCLIError checks does the error exist, and if it does, prints the error and exits the program.
func HandleCLIError(err *CLIError) {
	if err != nil {
		err.Print()
		os.Exit(1)
	}
}
