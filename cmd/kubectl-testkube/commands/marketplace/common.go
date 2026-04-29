// Package marketplace contains CLI subcommands under `testkube marketplace`
// for browsing, inspecting, and installing TestWorkflows published in the
// testkube-marketplace GitHub repository.
package marketplace

import (
	"github.com/kubeshop/testkube/pkg/marketplace"
)

// NewClient returns a marketplace client configured for production use.
// Centralized so individual commands can swap it out uniformly in the future
// (e.g. for caching or custom transports).
func NewClient() *marketplace.Client {
	return marketplace.NewClient()
}
