package common

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

type LocalCLIState int

const (
	LocalCLIConnected LocalCLIState = iota
	LocalCLIOtherLogin
	LocalCLIFailed
)

// An empty context may be an OSS user's kubeconfig mode.
func canUseLocalContext(cfg config.Data, firstRun bool) bool {
	return firstRun || isLocalContext(cfg)
}

// A var, so tests don't touch the real mark file.
var isOurCLIKey = localinstall.IsOurCLIKey

func isLocalContext(cfg config.Data) bool {
	c := cfg.CloudContext
	return cfg.ContextType == config.ContextTypeCloud && c.OrganizationId == localinstall.DemoOrgID &&
		strings.HasPrefix(c.ApiUri, "http://localhost:") && isOurCLIKey(c.ApiKey)
}

var errCLISettings = errors.New("the CLI settings can't be read or saved")

func ConnectLocalCLI(ctx context.Context, ports localinstall.Ports, email string, force bool) (LocalCLIState, error) {
	cfg, err := config.Load()
	if err != nil {
		return LocalCLIFailed, fmt.Errorf("%w: %v", errCLISettings, err)
	}
	if !force && !canUseLocalContext(cfg, !config.ExistedAtStart()) {
		return LocalCLIOtherLogin, nil
	}
	key := cfg.CloudContext.ApiKey
	if !isLocalContext(cfg) || !localinstall.CLIKeyWorks(ctx, ports, key) {
		if key, err = localinstall.CLIKey(ctx, ports, email); err != nil {
			return LocalCLIFailed, err
		}
	}
	if err := config.Save(localContext(cfg, ports, key)); err != nil {
		return LocalCLIFailed, fmt.Errorf("%w: %v", errCLISettings, err)
	}
	// Without it, the next run offers the hint instead.
	_ = localinstall.MarkCLIKey(key)
	return LocalCLIConnected, nil
}

// Raw errors carry paths and API replies; telemetry gets these.
func LocalCLIPlainError(err error) error {
	for _, plain := range []error{localinstall.ErrCLILogin, localinstall.ErrCLIKey, errCLISettings} {
		if errors.Is(err, plain) {
			return plain
		}
	}
	return errors.New("the CLI login failed")
}

// Fixed words for telemetry; raw errors can carry paths.
func LocalCLIFailure(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, localinstall.ErrCLILogin):
		return "login"
	case errors.Is(err, localinstall.ErrCLIKey):
		return "key"
	case errors.Is(err, errCLISettings):
		return "settings"
	}
	return "other"
}

// A context left pointing at nothing fails every command.
func DisconnectLocalCLI() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	cfg, ours := withoutLocalContext(cfg)
	if !ours || config.Save(cfg) != nil {
		return false
	}
	localinstall.ForgetCLIKey()
	return true
}

func withoutLocalContext(cfg config.Data) (config.Data, bool) {
	if !isLocalContext(cfg) {
		return cfg, false
	}
	cfg.ContextType, cfg.CloudContext = "", config.CloudContext{}
	return cfg, true
}

// Ports can move between runs, so every run rewrites it.
func localContext(cfg config.Data, ports localinstall.Ports, key string) config.Data {
	cfg.ContextType = config.ContextTypeCloud
	cfg.CloudContext = config.CloudContext{
		ApiUri:           fmt.Sprintf("http://localhost:%d", ports["api"]),
		UiUri:            fmt.Sprintf("http://localhost:%d", ports["dashboard"]),
		ApiKey:           key,
		TokenType:        config.TokenTypeAPI,
		OrganizationId:   localinstall.DemoOrgID,
		OrganizationName: "demo",
		EnvironmentId:    localinstall.DemoEnvID,
		EnvironmentName:  "my-first-environment",
	}
	return cfg
}

func NewInstallTracker() *telemetry.InstallTracker {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig
	}
	return telemetry.NewInstallTracker(telemetry.InstallTrackerConfig{
		Enabled:   cfg.TelemetryEnabled && !telemetry.DoNotTrack(),
		MachineID: telemetry.GetMachineID(),
		Version:   Version,
	})
}

// For local-trial errors outside step rows: PostHog, Segment, print.
func ExitOnLocalError(cmd *cobra.Command, event, stage string, cliErr *CLIError) {
	tracker := NewInstallTracker()
	tracker.Send(event, map[string]any{"stage": stage, "reason": string(cliErr.Code)})
	tracker.Wait()
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig
	}
	ExitOnCLIError(cmd, cfg, stage, cliErr)
}
