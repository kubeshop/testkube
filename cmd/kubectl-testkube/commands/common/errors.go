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

	// TKERR-12xx errors are related to configuration issues.

	// TKErrConfigInitFailed is returned when configuration init fails.
	TKErrConfigInitFailed ErrorCode = "TKERR-1201"
	// TKErrInvalidInstallConfig is returned when invalid configuration is supplied when installing or upgrading.
	TKErrInvalidInstallConfig ErrorCode = "TKERR-1202"
	// TKErrInvalidDockerConfig is returned when docker client configuration is invalid.
	TKErrInvalidDockerConfig ErrorCode = "TKERR-1203"
	// TKErrInvalidRuntimeParameter is returned when invalid runtime parameters are provided.
	TKErrInvalidRuntimeParameter ErrorCode = "TKERR-1204"

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

	// TKErrCleanOldMigrationJobFailed is returned in case of issues with old migration jobs.
	TKErrCleanOldMigrationJobFailed ErrorCode = "TKERR-1401"
)

const helpUrl = "https://testkubeworkspace.slack.com"

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
	pterm.DefaultHeader.Println("Testkube Init Error")

	pterm.DefaultSection.Println("Error Details")

	cmd := ""
	if e.ExecutedCommand != "" {
		pterm.FgDarkGray.Printfln("Executed command: %s", e.ExecutedCommand)
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
	pterm.DefaultBulletList.WithItems(items).Render()
	if cmd != "" {
		pterm.DefaultBox.Printfln("Error description is provided in context of binary execution %s", cmd)
	}

	pterm.Println()
	pterm.Println("Let us help you!")
	pterm.Printfln("Come say hi on Slack: %s", helpUrl)
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
