package common

import (
	"context"
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

// Never replace a login the user made, unless asked to.
func canUseLocalContext(cfg config.Data) bool {
	return cfg.ContextType == "" || isLocalContext(cfg)
}

func isLocalContext(cfg config.Data) bool {
	c := cfg.CloudContext
	return cfg.ContextType == config.ContextTypeCloud && c.OrganizationId == localinstall.DemoOrgID &&
		strings.HasPrefix(c.ApiUri, "http://localhost:")
}

func ConnectLocalCLI(ctx context.Context, ports localinstall.Ports, email string, force bool) LocalCLIState {
	cfg, err := config.Load()
	if err != nil {
		return LocalCLIFailed
	}
	if !force && !canUseLocalContext(cfg) {
		return LocalCLIOtherLogin
	}
	key := cfg.CloudContext.ApiKey
	if !isLocalContext(cfg) || !localinstall.CLIKeyWorks(ctx, ports, key) {
		if key, err = localinstall.CLIKey(ctx, ports, email); err != nil {
			return LocalCLIFailed
		}
	}
	if config.Save(localContext(cfg, ports, key)) != nil {
		return LocalCLIFailed
	}
	return LocalCLIConnected
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
