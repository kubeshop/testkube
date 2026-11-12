// Package controlplaneresources implements the CLI for Control Plane
// resources: `testkube apply`, and the per-kind `get` and `delete`
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

// ClientFactory builds the Control Plane client for a command. The commands
// take one, so a test can hand them a fake without changing shared state.
type ClientFactory func(cmd *cobra.Command) (client.Client, error)

// NewClient is the ClientFactory for the CLI's current Control Plane context.
func NewClient(cmd *cobra.Command) (client.Client, error) {
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
	headers, err := common.ResolveHeaders(cmd)
	if err != nil {
		return nil, err
	}
	return client.New(client.Config{
		URL:            cfg.CloudContext.ApiUri,
		Token:          cfg.CloudContext.ApiKey,
		OrganizationID: cfg.CloudContext.OrganizationId,
		EnvironmentID:  cfg.CloudContext.EnvironmentId,
		Headers:        headers,
		HTTPClient:     http.NewClient(common.ResolveSkipTLS(cmd, &cfg)),
	}), nil
}
