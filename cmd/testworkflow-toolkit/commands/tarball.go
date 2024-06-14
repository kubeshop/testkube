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
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/pkg/ui"
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

			for _, pair := range pairs {
				dirPath, url, found := strings.Cut(pair, "=")
				if !found {
					ui.Fail(fmt.Errorf("invalid tarball pair: %s", pair))
				}
				fmt.Printf("Downloading and unpacking %s to %s...\n", url, dirPath)

				// Start downloading the file
				resp, err := http.Get(url)
				ui.ExitOnError("download the tarball", err)

				if resp.StatusCode != http.StatusOK {
					ui.Fail(fmt.Errorf("failed to download the tarball: status code %d", resp.StatusCode))
				}

				// Process the files
				err = common.UnpackTarball(dirPath, resp.Body)
				if err != nil {
					ui.Fail(err)
				}
			}
		},
	}

	return cmd
}
