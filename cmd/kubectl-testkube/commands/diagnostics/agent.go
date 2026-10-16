package diagnostics

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/agent"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/ui"
)

func RegisterAgentValidators(cmd *cobra.Command, d diagnostics.Diagnostics) {
	namespace := cmd.Flag("namespace").Value.String()
	connect := func() (kubernetes.Interface, error) { return k8sclient.ConnectToK8s() }

	agentGroup := d.AddValidatorGroup("agent", nil)
	agentGroup.AddValidator(agent.NewValidator(connect, namespace))
}

func NewAgentCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"ag", "a"},
		Short:   "Diagnose the Testkube agent installed in the cluster",
		Run:     RunAgentCheckFunc(),
	}

	return cmd
}

func RunAgentCheckFunc() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		d := New(cmd)
		RegisterAgentValidators(cmd, d)

		err := d.Run()
		ui.ExitOnError("Running validations", err)
		ui.NL(2)
	}
}
