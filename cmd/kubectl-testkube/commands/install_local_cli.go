package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/localinstall"
)

// Never replace a login the user made themselves.
func canUseLocalContext(cfg config.Data) bool {
	return cfg.ContextType == "" || isLocalContext(cfg)
}

func isLocalContext(cfg config.Data) bool {
	c := cfg.CloudContext
	return cfg.ContextType == config.ContextTypeCloud && c.OrganizationId == localinstall.DemoOrgID &&
		strings.HasPrefix(c.ApiUri, "http://localhost:")
}

func connectCLI(ctx context.Context, ports localinstall.Ports, email string) bool {
	cfg, err := config.Load()
	if err != nil || !canUseLocalContext(cfg) {
		return false
	}
	key := cfg.CloudContext.ApiKey
	if !isLocalContext(cfg) || !localinstall.CLIKeyWorks(ctx, ports, key) {
		if key, err = localinstall.CLIKey(ctx, ports, email); err != nil {
			return false
		}
	}
	return config.Save(localContext(cfg, ports, key)) == nil
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
