// Package controlplaneresources implements the CLI for Control Plane
// resources - Insights boards, with AI agents and test
// bundles to come: `testkube apply`, and the per-kind `get` and `delete`
// subcommands. The rules are in pkg/controlplane/declarative, shared with the
// GitOps agent and the Control Plane.
package controlplaneresources

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative/client"
	"github.com/kubeshop/testkube/pkg/http"
)

// NewClientFn builds the Control Plane client. Tests replace it.
var NewClientFn = newClient

func newClient(cmd *cobra.Command) (client.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg.ContextType != config.ContextTypeCloud {
		return nil, errors.New("this command needs a Testkube Control Plane context; run `testkube login` first")
	}
	// GetClient refreshes an expired user token and stores the new one.
	if _, _, err := common.GetClient(cmd); err != nil {
		return nil, err
	}
	if cfg, err = config.Load(); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return client.New(client.Config{
		URL:            cfg.CloudContext.ApiUri,
		Token:          cfg.CloudContext.ApiKey,
		OrganizationID: cfg.CloudContext.OrganizationId,
		EnvironmentID:  cfg.CloudContext.EnvironmentId,
		HTTPClient:     http.NewClient(common.ResolveSkipTLS(cmd, &cfg)),
	}), nil
}
