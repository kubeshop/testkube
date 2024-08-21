package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer <pathUrlPairs>",
		Short: "Send files as tarball",

		Run: func(cmd *cobra.Command, pairs []string) {
			if len(pairs) == 0 {
				fmt.Println("nothing to send")
				os.Exit(0)
			}

			for _, pair := range pairs {
				dirPath, patternsAndUrl, found := strings.Cut(pair, ":")
				if !found {
					ui.Fail(fmt.Errorf("invalid files request: %s", pair))
				}
				patternsStr, url, found := strings.Cut(patternsAndUrl, "=")
				if !found {
					ui.Fail(fmt.Errorf("invalid files request: %s", pair))
				}
				patterns := strings.Split(patternsStr, ",")
				if len(patterns) == 0 {
					patterns = []string{"**/*"}
				}
				fmt.Printf("Packing and sending %s to %s...\n", dirPath, url)

				// Start downloading the file
				reader, writer := io.Pipe()
				go func() {
					err := common.WriteTarball(writer, dirPath, patterns)
					_ = writer.Close()
					ui.ExitOnError("write the tarball stream", err)
				}()
				resp, err := http.Post(url, "application/tar+gzip", reader)
				ui.ExitOnError("send the tarball request", err)
				_ = resp.Body.Close()

				if resp.StatusCode != http.StatusNoContent {
					ui.Fail(fmt.Errorf("failed to send the tarball: status code %d", resp.StatusCode))
				}
			}
		},
	}

	return cmd
}
