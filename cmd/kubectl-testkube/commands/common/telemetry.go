package common

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

// ExitOnCLIError reports the failure to telemetry and then prints the rich
// error details and terminates the command. It is a no-op for a nil error, so
// it can be called directly on a helper's returned *CLIError.
func ExitOnCLIError(cmd *cobra.Command, clientCfg config.Data, errType string, cliErr *CLIError) {
	if cliErr == nil {
		return
	}

	SendErrTelemetry(cmd, clientCfg, errType, cliErr)
	HandleCLIError(cliErr)
}

// SendErrTelemetry reports a command failure. It is a no-op when the user has
// telemetry disabled.
func SendErrTelemetry(cmd *cobra.Command, clientCfg config.Data, errType string, errorLogs error) {
	if !clientCfg.TelemetryEnabled {
		return
	}

	var errorStackTrace = fmt.Sprintf("%+v", errorLogs)
	// Carry the TKERR code over when we know it, so the failure is groupable.
	var errCode string
	var cliErr *CLIError
	if errors.As(errorLogs, &cliErr) {
		errCode = string(cliErr.Code)
		errorStackTrace = cliErr.StackTrace
	}

	ui.Debug("collecting anonymous telemetry data, you can disable it by calling `testkube disable telemetry`")
	out, err := telemetry.SendCmdErrorEventWithLicense(cmd, Version, errType, errorStackTrace, "", "", errCode)
	if ui.Verbose && err != nil {
		ui.Err(err)
	}

	ui.Debug("telemetry send event response", out)
}

// SendAttemptTelemetry reports that a command was started. It is a no-op when
// the user has telemetry disabled.
func SendAttemptTelemetry(cmd *cobra.Command, clientCfg config.Data) {
	if !clientCfg.TelemetryEnabled {
		return
	}

	ui.Debug("collecting anonymous telemetry data, you can disable it by calling `testkube disable telemetry`")
	out, err := telemetry.SendCmdAttemptEvent(cmd, Version, TelemetryUserID(cmd, &clientCfg))
	if ui.Verbose && err != nil {
		ui.Err(err)
	}

	ui.Debug("telemetry send event response", out)
}
