package commands

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	TarballRetryMaxAttempts = 5
)

func NewTarballCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tarball <pathUrlPairs>",
		Short: "Download and unpack tarball file(s)",

		Run: func(cmd *cobra.Command, pairs []string) {
			if len(pairs) == 0 {
				fmt.Println("nothing to fetch and unpack")
				os.Exit(0)
			}

			processPair := func(pair string) {
				dirPath, url, found := strings.Cut(pair, "=")
				if !found {
					ui.Fail(fmt.Errorf("invalid tarball pair: %s", pair))
				}
				fmt.Printf("Downloading and unpacking %s to %s...\n", url, dirPath)

				// Start downloading the file
				attempt := 1
				for {
					resp, err := http.Get(url)
					if err == nil && resp.StatusCode != http.StatusOK {
						err = fmt.Errorf("status code %d", resp.StatusCode)
					}
					if err == nil {
						// Process the files
						err = common.UnpackTarball(dirPath, resp.Body)
						resp.Body.Close()
						if err == nil {
							break
						}
						fmt.Printf("failed to unpack the tarball: %s\n", err.Error())
					} else {
						resp.Body.Close()
						fmt.Printf("failed to download the tarball: %s\n", err.Error())
					}

					// Retry when it is possible
					if TarballRetryMaxAttempts == attempt {
						os.Exit(1)
					}
					attempt++
					fmt.Printf("retrying - attempt %d/%d.\n", attempt, TarballRetryMaxAttempts)
				}
			}

			for _, pair := range pairs {
				processPair(pair)
			}
		},
	}

	return cmd
}
