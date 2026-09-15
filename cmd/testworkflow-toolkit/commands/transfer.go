package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
)

const (
	transferRetryCount = 3
)

var transferRetryBaseDelay = 500 * time.Millisecond

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

	var lastErr error
	for attempt := 0; attempt < transferRetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * transferRetryBaseDelay)
			fmt.Fprintf(output, "retrying tarball transfer (attempt %d/%d)...\n", attempt+1, transferRetryCount)
		}
		retryable, err := sendTarballOnce(dirPath, patterns, url)
		if err == nil {
			return 0
		}
		lastErr = err
		if !retryable {
			fmt.Fprintf(output, "error: %s\n", err.Error())
			return 1
		}
	}
	fmt.Fprintf(output, "error: tarball transfer failed after %d attempts: %s\n", transferRetryCount, lastErr.Error())
	return 1
}

func sendTarballOnce(dirPath string, patterns []string, url string) (retryable bool, err error) {
	errChan := make(chan error, 1)
	reader, writer := io.Pipe()
	go func() {
		e := common.WriteTarball(writer, dirPath, patterns)
		_ = writer.Close()
		errChan <- e
	}()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, reader)
	if err != nil {
		return false, fmt.Errorf("create the tarball request - %w", err)
	}
	req.Header.Set("Content-Type", "application/tar+gzip")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return true, fmt.Errorf("send the tarball request - %w", err)
	}
	defer resp.Body.Close()

	if writeErr := <-errChan; writeErr != nil {
		return false, fmt.Errorf("write the tarball stream - %w", writeErr)
	}

	if resp.StatusCode == http.StatusNoContent {
		return false, nil
	}
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return true, fmt.Errorf("failed to send the tarball: status code %d", resp.StatusCode)
	}
	return false, fmt.Errorf("failed to send the tarball: status code %d", resp.StatusCode)
}
