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
	// TKErrResourceLookupFailed is returned when listing or reading resources from the Kubernetes cluster fails.
	// The lookup itself did not complete, so the caller cannot tell whether the resource exists: a resource that
	// answered and is absent uses TKErrResourceNotFound.
	TKErrResourceLookupFailed ErrorCode = "TKERR-1702"

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
	// TKErrAPIClientInitFailed is returned when the Testkube API client can't be built for a command.
	// GetClient fails on the '--header' flag, on the config file, on building the client for the '--client'
	// type, and for a cloud context also on refreshing the stored token, on the re-login it falls back to,
	// and on writing the new token back. On the default proxy client the kubeconfig is the likely cause and
	// the token paths are unreachable, so the hint names the cluster before the credentials.
	TKErrAPIClientInitFailed ErrorCode = "TKERR-1807"

	// TKERR-19xx errors are related to the resource commands that talk to the Testkube API.

	// TKErrAPIReadFailed is returned when reading one resource, or listing resources, through the Testkube
	// API fails. One code covers get and list for the reason TKErrAgentWriteFailed gives: the client
	// helpers share one preamble and differ only in the final call, so the code says a read was attempted
	// and the title says which. A resource the API answered about and that is absent uses
	// TKErrResourceNotFound.
	TKErrAPIReadFailed ErrorCode = "TKERR-1901"
	// TKErrAPIWriteFailed is returned when creating, updating or deleting a resource through the Testkube
	// API fails. Reads use TKErrAPIReadFailed.
	TKErrAPIWriteFailed ErrorCode = "TKERR-1902"
	// TKErrOutputRenderFailed is returned when a command got its result but could not print it: an unusable
	// '--output' type or '--go-template' expression, a value that will not marshal, or the CRD template
	// behind '--crd-only'. The data is good and only the presentation failed, so the hint points at the
	// output flags rather than at the fetch.
	TKErrOutputRenderFailed ErrorCode = "TKERR-1903"
)

const helpUrl = "https://testkubeworkspace.slack.com"

// ConfigFileHint is the recovery hint for any failure to read or write
// the CLI config file.
const ConfigFileHint = "Check is the Testkube config file (~/.testkube/config.json) accessible and has right permissions"

// AgentLookupHint is the recovery hint for a failure to read an agent from the
// control plane. The read fails on the name, on the credentials, or on the
// connection, so the hint names the first two and a command that lists what
// exists.
const AgentLookupHint = "Check the agent name or ID and that your credentials are valid, or list the agents with `testkube get agents`"

// AgentWriteHint is the recovery hint for a failure to create, update or delete
// an agent on the control plane. The preamble is the same as a read, so what is
// left to check is the permission to change it.
const AgentWriteHint = "Check that your credentials are valid and that your user can manage the agents of this organization"

// ClusterLookupHint is the recovery hint for a failure to read namespaces, pods
// or CRDs from the cluster. It names the kubeconfig, because the CLI reads the
// cluster with the same context kubectl uses.
const ClusterLookupHint = "Check that your kubeconfig points at the right cluster and that you can read it, for example with `kubectl get namespaces`"

// APIClientHint is the recovery hint for a failure to build the Testkube API
// client. The client is built from the current context and the stored token, so
// those are the two things to look at.
const APIClientHint = "Check that your kubeconfig points at the right cluster, or sign in again with `testkube pro login` if you use a cloud context and your token has expired"

// APIReadHint is the recovery hint for a failed read of resources through the
// Testkube API. It does not name a command that lists them: the listing is
// usually the command that just failed.
const APIReadHint = "Check that your credentials are valid and that the current context points at the organization and environment you expect"

// APIWriteHint is the recovery hint for a failed create, update or delete
// through the Testkube API. A write fails on the same things a read does, so
// what is left to check is the permission to change the resource.
const APIWriteHint = "Check that your credentials are valid and that your user can manage the resources of this environment or namespace"

// APIDeleteHint is the recovery hint for a failed delete. A delete is the one
// write where the resource being gone already is a normal outcome, which is what
// '--ignore-not-found' is for.
const APIDeleteHint = "Check the name or the '--label' selector, or pass '--ignore-not-found' to succeed when the resource is already gone"

// OutputRenderHint is the recovery hint for a command that fetched its result
// and could not print it. The fetch worked, so the flags that shape the output
// are what is left.
const OutputRenderHint = "Check the '--output' value (pretty, json, yaml or go) and the '--go-template' expression"

// NameFlagHint is the recovery hint for a command that needs '--name' and did
// not get one.
const NameFlagHint = "Pass the name with the '--name' flag"

// NameOrSelectorHint is the recovery hint for a command that accepts either a
// name or a label selector and got neither.
const NameOrSelectorHint = "Pass the name as an argument, or select by labels with '--label', for example '--label app=backend'"

// NameConflictHint is the recovery hint for a create that found the name taken.
const NameConflictHint = "Choose a name that is free, or pass '--update' to overwrite the existing one"

// BoolFlagValueHint is the recovery hint for a boolean flag that could not be
// parsed.
const BoolFlagValueHint = "Check the flag value; a boolean flag takes true or false, or drop the flag to use its default"

// WebhookFlagsHint is the recovery hint for a failure to turn the webhook flags
// into API options. Each of these flags carries structured text the CLI parses,
// which is where such a failure comes from.
const WebhookFlagsHint = "Check the '--events', '--header', '--config' and '--parameter' values, or that '--payload-template' points at a readable file"

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
