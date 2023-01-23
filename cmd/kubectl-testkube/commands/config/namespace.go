package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewConfigureNamespaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "namespace <value>",
		Short: "Set namespace for testkube client",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please pass valid namespace name")
			}

			errors := validation.IsDNS1123Subdomain(args[0])
			if len(errors) > 0 {
				return fmt.Errorf("invalid name, errors: %v", errors)
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.Namespace = args[0]
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New namespace set to", cfg.Namespace)
		},
	}

	return cmd
}
