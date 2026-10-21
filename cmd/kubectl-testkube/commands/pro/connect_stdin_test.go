package pro

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/agents"
)

// `pro connect` always asks for confirmation, which needs the terminal that piped input takes,
// so only `agents install` offers reading the global template from stdin.
func TestGlobalTemplatePath_OffersStdinOnlyWhereItWorks(t *testing.T) {
	assert.Contains(t, agents.NewInstallAgentCommand().Flags().Lookup("global-template-path").Usage, "stdin")
	assert.NotContains(t, NewConnectCmd().Flags().Lookup("global-template-path").Usage, "stdin")
}
