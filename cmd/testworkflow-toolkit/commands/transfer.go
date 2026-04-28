package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
)

func NewTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer <pathUrlPairs>",
		Short: "Send files as tarball",

		Run: func(cmd *cobra.Command, pairs []string) {
			exitCode := ProcessTransfers(pairs, cmd.OutOrStdout())
			os.Exit(exitCode)
		},
	}

	return cmd
}

// ProcessTransfers handles multiple transfer pairs
// Returns 0 if all succeed, 1 if any fail
func ProcessTransfers(pairs []string, output io.Writer) int {
	if len(pairs) == 0 {
		fmt.Fprintln(output, "nothing to send")
		return 0
	}

	for _, pair := range pairs {
		if exitCode := ProcessTransferPair(pair, output); exitCode != 0 {
			return exitCode
		}
	}
	return 0
}

// ProcessTransferPair sends a single tarball to the specified URL
// Returns 0 on success, 1 on failure
func ProcessTransferPair(pair string, output io.Writer) int {
	dirPath, patternsAndUrl, found := strings.Cut(pair, ":")
	if !found {
		fmt.Fprintf(output, "error: invalid files request: %s\n", pair)
		return 1
	}
	patternsStr, url, found := strings.Cut(patternsAndUrl, "=")
	if !found {
		fmt.Fprintf(output, "error: invalid files request: %s\n", pair)
		return 1
	}
	patterns := strings.Split(patternsStr, ",")
	if len(patterns) == 0 || (len(patterns) == 1 && patterns[0] == "") {
		patterns = []string{"**/*"}
	}
	fmt.Fprintf(output, "Packing and sending %s to %s...\n", dirPath, url)

	// Create a channel to capture errors from goroutine
	errChan := make(chan error, 1)

	// Start packing the files
	reader, writer := io.Pipe()
	go func() {
		err := common.WriteTarball(writer, dirPath, patterns)
		_ = writer.Close()
		if err != nil {
			errChan <- err
		} else {
			errChan <- nil
		}
	}()

	// Send the tarball
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, reader)
	if err != nil {
		fmt.Fprintf(output, "error: create the tarball request - %s\n", err.Error())
		return 1
	}
	req.Header.Set("Content-Type", "application/tar+gzip")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(output, "error: send the tarball request - %s\n", err.Error())
		return 1
	}
	defer resp.Body.Close()

	// Check for write errors
	writeErr := <-errChan
	if writeErr != nil {
		fmt.Fprintf(output, "error: write the tarball stream - %s\n", writeErr.Error())
		return 1
	}

	if resp.StatusCode != http.StatusNoContent {
		fmt.Fprintf(output, "error: failed to send the tarball: status code %d\n", resp.StatusCode)
		return 1
	}

	return 0
}
