package artifacts

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	executionID string
	filename    string
	destination string
	downloadDir string
)

func NewListArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <executionName>",
		Aliases: []string{"artifacts"},
		Short:   "List artifacts of the given test or test suite execution name",
		Args:    validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			executionID = args[0]
			cmd.SilenceUsage = true
			client, _ := common.GetClient(cmd)
			execution, err := client.GetExecution(executionID)
			var artifacts testkube.Artifacts
			var errArtifacts error
			if err == nil && execution.Id != "" {
				artifacts, errArtifacts = client.GetExecutionArtifacts(executionID)
				ui.ExitOnError("getting test artifacts ", errArtifacts)
			} else {
				_, err := client.GetTestSuiteExecution(executionID)
				ui.ExitOnError("no test or test suite execution was found with the following id", err)
				artifacts, errArtifacts = client.GetTestSuiteExecutionArtifacts(executionID)
				ui.ExitOnError("getting test suite artifacts ", errArtifacts)
			}

			ui.Table(artifacts, os.Stdout)
		},
	}

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")

	// output renderer flags
	return cmd
}

func NewDownloadSingleArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "single <executionName> <fileName> <destinationDir>",
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
		Use:   "all <executionName>",
		Short: "download artifacts",
		Args:  validator.ExecutionName,
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
