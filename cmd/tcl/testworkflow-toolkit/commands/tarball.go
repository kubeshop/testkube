// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	relativeCheckRe = regexp.MustCompile(`(^|/)\.\.(/|$)`)
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
				uncompressedStream, err := gzip.NewReader(resp.Body)
				ui.ExitOnError("start reading gzip", err)
				tarReader := tar.NewReader(uncompressedStream)

				// Unpack them
				for {
					header, err := tarReader.Next()
					if err == io.EOF {
						break
					}
					ui.ExitOnError("get next entry from tarball", err)
					if filepath.IsAbs(header.Name) || relativeCheckRe.MatchString(filepath.ToSlash(header.Name)) {
						ui.Fail(fmt.Errorf("unsafe file path in the tarball: %s", header.Name))
					}

					filePath := filepath.Join(dirPath, header.Name)

					switch header.Typeflag {
					case tar.TypeDir:
						err := os.Mkdir(filePath, 0755)
						ui.ExitOnError(fmt.Sprintf("%s: create directory", filePath), err)
					case tar.TypeReg:
						err := os.MkdirAll(filepath.Dir(filePath), 0755)
						ui.ExitOnError(fmt.Sprintf("%s: create directory tree", filePath), err)
						outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
						ui.ExitOnError(fmt.Sprintf("%s: create file", filePath), err)
						_, err = io.Copy(outFile, tarReader)
						ui.ExitOnError(fmt.Sprintf("%s: write file", filePath), err)
						outFile.Close()
					case tar.TypeSymlink:
						err := os.MkdirAll(filepath.Dir(filePath), 0755)
						ui.ExitOnError(fmt.Sprintf("%s: create directory tree", filePath), err)
						err = os.Symlink(header.Linkname, filePath)
						ui.ExitOnError(fmt.Sprintf("%s: create symlink", filePath), err)
					default:
						ui.Fail(fmt.Errorf("unknown entry type in the transferred archive: '%x' in %s", header.Typeflag, filePath))
					}
				}
				resp.Body.Close()
			}
		},
	}

	return cmd
}
