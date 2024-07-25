package common

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

const (
	// TKERR-1xx errors are to issues when running testkube CLI commands.

	// TKERR-11xx errors are related to missing dependencies.

	// TKErrMissingDependencyHelm is returned when kubectl is not found in $PATH.
	TKErrMissingDependencyHelm = "TKERR-1101"
	// TKErrMissingDependencyKubectl is returned when kubectl is not found in $PATH.
	TKErrMissingDependencyKubectl = "TKERR-1102"

	// TKERR-12xx errors are related to configuration issues.

	// TKErrConfigLoadingFailed is returned when configuration loading fails.
	TKErrConfigLoadingFailed = "TKERR-1201"
	// TKErrInvalidInstallConfig is returned when invalid configuration is supplied when installing or upgrading.
	TKErrInvalidInstallConfig = "TKERR-1202"

	// TKERR-13xx errors are related to install operations.

	// TKErrHelmCommandFailed is returned when a helm command fails.
	TKErrHelmCommandFailed = "TKERR-1301"
	// TKErrKubectlCommandFailed is returned when a kubectl command fail.
	TKErrKubectlCommandFailed = "TKERR-1302"
)

type CLIError struct {
	Code        string
	Title       string
	Description string
	ActualError error
	StackTrace  string
	MoreInfo    string
}

func (e *CLIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Description)
}

func (e *CLIError) Print() {
	pterm.DefaultHeader.Println("Testkube Init Error")

	pterm.DefaultSection.Println("Error Details")

	items := []pterm.BulletListItem{
		{Level: 0, Text: pterm.Sprintf("[%s]: %s", e.Code, e.Title), TextStyle: pterm.NewStyle(pterm.FgRed)},
		{Level: 0, Text: pterm.Sprintf("%s", e.Description), TextStyle: pterm.NewStyle(pterm.FgLightWhite)},
	}
	if e.MoreInfo != "" {
		items = append(items, pterm.BulletListItem{Level: 0, Text: pterm.Sprintf("%s", e.MoreInfo), TextStyle: pterm.NewStyle(pterm.FgGray)})
	}
	pterm.DefaultBulletList.WithItems(items).Render()
}

func NewCLIError(code, title, moreInfoURL string, err error) *CLIError {
	return &CLIError{
		Code:        code,
		Title:       title,
		Description: err.Error(),
		ActualError: err,
		MoreInfo:    moreInfoURL,
	}
}

// HandleCLIError checks does the error exist, and if it does, prints the error and exits the program.
func HandleCLIError(err *CLIError) {
	if err != nil {
		err.Print()
		os.Exit(1)
	}
}
