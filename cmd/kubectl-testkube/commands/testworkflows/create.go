package testworkflows

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/marketplace"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateTestWorkflowCmd() *cobra.Command {
	var (
		name     string
		filePath string
		url      string
		update   bool
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:     "testworkflow",
		Aliases: []string{"testworkflows", "tw"},
		Args:    cobra.MaximumNArgs(0),
		Short:   "Create test workflow",

		Run: func(cmd *cobra.Command, _ []string) {
			if filePath != "" && url != "" {
				ui.Failf("--file and --url are mutually exclusive")
			}

			var raw []byte
			switch {
			case url != "":
				client := marketplace.NewClient()
				body, err := client.FetchURL(cmd.Context(), url)
				ui.ExitOnError("fetching test workflow from "+url, err)
				raw = body
			case filePath != "":
				file, err := os.Open(filePath)
				ui.ExitOnError("reading "+filePath+" file", err)
				defer file.Close()
				body, err := io.ReadAll(file)
				ui.ExitOnError("reading input", err)
				raw = body
			default:
				fi, err := os.Stdin.Stat()
				ui.ExitOnError("reading stdin", err)
				if fi.Mode()&os.ModeDevice != 0 {
					ui.Failf("you need to pass stdin, --file, or --url argument")
				}
				body, err := io.ReadAll(cmd.InOrStdin())
				ui.ExitOnError("reading input", err)
				raw = body
			}

			CreateOrUpdateFromBytes(cmd, raw, name, update, dryRun)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "test workflow name")
	cmd.Flags().BoolVar(&update, "update", false, "update, if test workflow already exists")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "file path to get the test workflow specification")
	cmd.Flags().StringVarP(&url, "url", "u", "", "URL to fetch the test workflow specification from (mutually exclusive with --file)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate test workflow specification yaml")
	cmd.Flags().MarkDeprecated("disable-webhooks", "disable-webhooks is deprecated")

	return cmd
}
