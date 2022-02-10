package commands

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewMigrateCmd() *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "manual migrate command",
		Long:  `migrate command manages migrations will run migrations greater or equals current version`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			RunMigrations(cmd)
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "testkube namespace")

	return cmd
}
