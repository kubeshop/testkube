package artifacts

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	executionID string
	filename    string
	destination string
	downloadDir string
)

func NewListArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <executionID>",
		Aliases: []string{"artifacts"},
		Short:   "List artifacts of the given execution ID",
		Args:    validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			executionID = args[0]
			cmd.SilenceUsage = true
			client, _ := common.GetClient(cmd)
			artifacts, err := client.GetExecutionArtifacts(executionID)
			ui.ExitOnError("getting artifacts ", err)

			ui.Table(artifacts, os.Stdout)
		},
	}

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")

	// output renderer flags
	return cmd
}

func NewDownloadSingleArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "single <executionID> <fileName> <destinationDir>",
		Short: "download artifact",
		Args:  validator.ExecutionIDAndFileNames,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]
			filename := args[1]
			destination := args[2]

			client, _ := common.GetClient(cmd)
			f, err := client.DownloadFile(executionID, filename, destination)
			ui.ExitOnError("downloading file"+filename, err)

			ui.Info(fmt.Sprintf("File %s downloaded.\n", f))
		},
	}

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")
	cmd.PersistentFlags().StringVarP(&filename, "filename", "f", "", "name of the file")
	cmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "name of the file")

	// output renderer flags
	return cmd
}

func NewDownloadAllArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all <executionID>",
		Short: "download artifacts",
		Args:  validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]
			client, _ := common.GetClient(cmd)
			tests.DownloadArtifacts(executionID, downloadDir, client)
		},
	}

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")

	// output renderer flags
	return cmd
}
