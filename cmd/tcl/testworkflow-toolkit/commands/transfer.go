// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
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

				if resp.StatusCode != http.StatusNoContent {
					ui.Fail(fmt.Errorf("failed to send the tarball: status code %d", resp.StatusCode))
				}
			}
		},
	}

	return cmd
}
