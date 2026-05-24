package commands

import (
	"github.com/kubeshop/testkube/pkg/checker"
	"github.com/spf13/cobra"
)

var (
	outputFormat  string
	ignoreBlocker bool
)

func NewCheckCMD() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "check",
		Aliases: []string{"check", "ch"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Run diagnostic checks on your Testkube installation",
		Run: func(cmd *cobra.Command, args []string) {
			CheckSuite := []checker.CheckSuiteName{
				checker.ClusterCheck,
				checker.TestkubeAPICheck,
				checker.TestkubePermissionCheck,
			}
			sc := checker.NewSystemChecker(CheckSuite)
			success := sc.ExecuteSuite(ignoreBlocker)
			checker.PrintFinalResults(success, sc.SuiteResults, outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format. One of: table, json")
	cmd.Flags().BoolVar(&ignoreBlocker, "ignore-blocker", false, "continue running all checks even if a blocker fails")
	return cmd
}
