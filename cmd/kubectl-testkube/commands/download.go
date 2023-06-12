package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	executionID string
	filename    string
	destination string
	downloadDir string
	format      string
	masks       []string
)

func NewDownloadCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "download <resource>",
		Short: "Artifacts management commands",
		Args:  validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		}}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "should I show additional debug messages")

	cmd.AddCommand(NewDownloadSingleArtifactsCmd())
	cmd.AddCommand(NewDownloadAllArtifactsCmd())

	return cmd
}

func NewListArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <executionName>",
		Short: "List artifacts of the given execution name",
		Args:  validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			executionID = args[0]
			cmd.SilenceUsage = true
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			artifacts, err := client.GetExecutionArtifacts(executionID)
			ui.ExitOnError("getting artifacts ", err)

			ui.Table(artifacts, os.Stdout)
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "should I show additional debug messages")

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")

	// output renderer flags
	return cmd
}

func NewDownloadSingleArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact <executionName> <fileName> <destinationDir>",
		Short: "download artifact",
		Args:  validator.ExecutionIDAndFileNames,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]
			filename := args[1]
			destination := args[2]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			f, err := client.DownloadFile(executionID, filename, destination)
			ui.ExitOnError("downloading file"+filename, err)

			ui.Info(fmt.Sprintf("File %s downloaded.\n", f))
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "should I show additional debug messages")

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")
	cmd.PersistentFlags().StringVarP(&filename, "filename", "f", "", "name of the file")
	cmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "name of the file")

	// output renderer flags
	return cmd
}

func NewDownloadAllArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifacts <executionName>",
		Aliases: []string{"a"},
		Short:   "download artifacts",
		Args:    validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			tests.DownloadArtifacts(executionID, downloadDir, format, masks, client)
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "should I show additional debug messages")

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")
	cmd.Flags().StringVar(&format, "format", "folder", "data format for storing files, one of folder|archive")
	cmd.Flags().StringArrayVarP(&masks, "mask", "", []string{}, "regexp to filter downloaded files, single or comma separated, like report/.* or .*\\.json,.*\\.js$")

	// output renderer flags
	return cmd
}
