package commands

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/scripts"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	executionID string
	filename    string
	destination string
	downloadDir string
)

func NewArtifactsCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "artifacts",
		Short: "Artifacts management commands",
		Args:  validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// version validation
			// if client version is less than server version show warning
			client, _ := common.GetClient(cmd)

			err := ValidateVersions(client)
			if err != nil {
				ui.Warn(err.Error())
			}
		},
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")

	cmd.AddCommand(NewListArtifactsCmd())
	cmd.AddCommand(NewDownloadSingleArtifactsCmd())
	cmd.AddCommand(NewDownloadAllArtifactsCmd())
	// output renderer flags
	return cmd
}

func NewListArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <executionID>",
		Short: "List artifacts of the given execution ID",
		Args:  validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			executionID = args[0]
			cmd.SilenceUsage = true
			client, _ := common.GetClient(cmd)
			artifacts, err := client.GetExecutionArtifacts(executionID)
			ui.ExitOnError("getting artifacts ", err)

			ui.Table(artifacts, os.Stdout)
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")

	// output renderer flags
	return cmd
}

func NewDownloadSingleArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download-one",
		Short: "download artifact",
		Args:  validator.ExecutionID,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if executionID == "" {
				cmd.SilenceUsage = true
				return fmt.Errorf("execution-id is a required parameter")
			}

			if filename == "" {
				cmd.SilenceUsage = true
				return fmt.Errorf("fileName is a required parameter")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client, _ := common.GetClient(cmd)
			if f, err := client.DownloadFile(executionID, filename, destination); err != nil {
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
	cmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "name of the file")

	// output renderer flags
	return cmd
}

func NewDownloadAllArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "download artifact",
		Args:  validator.ExecutionID,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if executionID == "" {
				cmd.SilenceUsage = true
				return fmt.Errorf("execution-id is a required parameter")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _ := common.GetClient(cmd)
			scripts.DownloadArtifacts(executionID, downloadDir, client)
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")

	// output renderer flags
	return cmd
}
