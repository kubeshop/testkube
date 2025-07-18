package commands

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
)

const (
	// TarballRetryMaxAttempts defines the maximum number of download attempts before giving up
	TarballRetryMaxAttempts = 5
)

var (
	// ErrInvalidPairFormat is returned when the tarball pair format is invalid
	ErrInvalidPairFormat = errors.New("invalid tarball pair format, expected: path=url")
)

func NewTarballCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tarball <pathUrlPairs>",
		Short: "Download and unpack tarball file(s)",

		Run: func(cmd *cobra.Command, pairs []string) {
			exitCode := ProcessTarballs(pairs, cmd.OutOrStdout())
			os.Exit(exitCode)
		},
	}

	return cmd
}

// ProcessTarballs handles multiple tarball pairs
// Returns 0 if all succeed, 1 if any fail
func ProcessTarballs(pairs []string, output io.Writer) int {
	if len(pairs) == 0 {
		fmt.Fprintln(output, "nothing to fetch and unpack")
		return 0
	}

	for _, pair := range pairs {
		if exitCode := ProcessTarballPair(pair, output); exitCode != 0 {
			return exitCode
		}
	}
	return 0
}

// ProcessTarballPair downloads and unpacks a single tarball
// Returns 0 on success, 1 on failure
func ProcessTarballPair(pair string, output io.Writer) int {
	dirPath, url, found := strings.Cut(pair, "=")
	if !found {
		fmt.Fprintf(output, "error: %v - %s\n", ErrInvalidPairFormat, pair)
		return 1
	}
	fmt.Fprintf(output, "Downloading and unpacking %s to %s...\n", url, dirPath)

	// Start downloading the file
	for attempt := 1; attempt <= TarballRetryMaxAttempts; attempt++ {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("status code %d", resp.StatusCode)
		}

		if err == nil {
			// Process the files
			err = common.UnpackTarball(dirPath, resp.Body)
			resp.Body.Close()
			if err == nil {
				return 0
			}
			fmt.Fprintf(output, "failed to unpack the tarball: %s\n", err.Error())
		} else {
			if resp != nil {
				resp.Body.Close()
			}
			fmt.Fprintf(output, "failed to download the tarball: %s\n", err.Error())
		}

		// Check if we should retry
		if attempt < TarballRetryMaxAttempts {
			fmt.Fprintf(output, "retrying - attempt %d/%d.\n", attempt+1, TarballRetryMaxAttempts)
		}
	}

	return 1
}
