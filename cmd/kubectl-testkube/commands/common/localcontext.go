package common

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/localinstall"
)

type LocalCLIState int

const (
	LocalCLIConnected LocalCLIState = iota
	LocalCLIOtherLogin
	LocalCLIFailed
)

// An empty context may be an OSS user's kubeconfig mode.
func canUseLocalContext(cfg config.Data, firstRun bool, ours keyCheck) bool {
	return firstRun || isLocalContext(cfg, ours)
}

// Passed in, so tests don't touch the real mark file.
type keyCheck func(key string) bool

func isLocalContext(cfg config.Data, ours keyCheck) bool {
	c := cfg.CloudContext
	return cfg.ContextType == config.ContextTypeCloud && c.OrganizationId == localinstall.DemoOrgID &&
		strings.HasPrefix(c.ApiUri, "http://localhost:") && ours(c.ApiKey)
}

var errCLISettings = errors.New("the CLI settings can't be read or saved")

func ConnectLocalCLI(ctx context.Context, ports localinstall.Ports, email string, force bool) (LocalCLIState, error) {
	cfg, err := config.Load()
	if err != nil {
		return LocalCLIFailed, fmt.Errorf("%w: %v", errCLISettings, err)
	}
	if !force && !canUseLocalContext(cfg, !config.ExistedAtStart(), localinstall.IsOurCLIKey) {
		return LocalCLIOtherLogin, nil
	}
	key := cfg.CloudContext.ApiKey
	if !isLocalContext(cfg, localinstall.IsOurCLIKey) || !localinstall.CLIKeyWorks(ctx, ports, key) {
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
	cfg, ours := withoutLocalContext(cfg, localinstall.IsOurCLIKey)
	if !ours || config.Save(cfg) != nil {
		return false
	}
	localinstall.ForgetCLIKey()
	return true
}

func withoutLocalContext(cfg config.Data, ours keyCheck) (config.Data, bool) {
	if !isLocalContext(cfg, ours) {
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
