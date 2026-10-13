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
func canUseLocalContext(cfg config.Data, firstRun bool) bool {
	return firstRun || isLocalContext(cfg)
}

func isLocalContext(cfg config.Data) bool {
	c := cfg.CloudContext
	return cfg.ContextType == config.ContextTypeCloud && c.OrganizationId == localinstall.DemoOrgID &&
		strings.HasPrefix(c.ApiUri, "http://localhost:")
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
	cfg, ours := withoutLocalContext(cfg)
	return ours && config.Save(cfg) == nil
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
