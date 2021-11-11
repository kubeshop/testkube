package commands

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/scripts"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	executionID string
	filename    string
)

func NewArtifactsCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "artifacts",
		Short: "Artifacts management commands",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if executionID == "" {
				cmd.SilenceUsage = true
				return fmt.Errorf("execution-id is a required parameter")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client, _ := scripts.GetClient(cmd)
			if filename == "" {
				artifacts, err := client.GetExecutionArtifacts(executionID)
				if err != nil {
					return err
				}
				ui.Table(artifacts, os.Stdout)
				return nil
			}

			if f, err := client.DownloadFile(executionID, filename); err != nil {
				cmd.SilenceUsage = true
				return err
			} else {
				fmt.Printf("File %s downloaded.\n", f)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")
	cmd.PersistentFlags().StringVarP(&filename, "filename", "f", "", "name of the file")

	// output renderer flags
	return cmd
}
