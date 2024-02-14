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
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/ui"
)

var directAddGzipEncoding = artifacts.WithMinioOptionsEnhancer(func(options *minio.PutObjectOptions, path string, size int64) {
	options.ContentType = "application/gzip"
	options.ContentEncoding = "gzip"
})

var directDisableMultipart = artifacts.WithMinioOptionsEnhancer(func(options *minio.PutObjectOptions, path string, size int64) {
	options.DisableMultipart = true
})

var directDetectMimetype = artifacts.WithMinioOptionsEnhancer(func(options *minio.PutObjectOptions, path string, size int64) {
	if options.ContentType == "" {
		options.ContentType = artifacts.DetectMimetype(path)
	}
})

var directUnpack = artifacts.WithMinioOptionsEnhancer(func(options *minio.PutObjectOptions, path string, size int64) {
	options.UserMetadata = map[string]string{
		"X-Amz-Meta-Snowball-Auto-Extract": "true",
		"X-Amz-Meta-Minio-Snowball-Prefix": env.WorkflowName() + "/" + env.ExecutionId(),
	}
})

var cloudAddGzipEncoding = artifacts.WithRequestEnhancerCloud(func(req *http.Request, path string, size int64) {
	req.Header.Set("Content-Type", "application/gzip")
	req.Header.Set("Content-Encoding", "gzip")
})

var cloudUnpack = artifacts.WithRequestEnhancerCloud(func(req *http.Request, path string, size int64) {
	req.Header.Set("X-Amz-Meta-Snowball-Auto-Extract", "true")
})

var cloudDetectMimetype = artifacts.WithRequestEnhancerCloud(func(req *http.Request, path string, size int64) {
	if req.Header.Get("Content-Type") == "" {
		contentType := artifacts.DetectMimetype(path)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if contentType == "application/gzip" && req.Header.Get("Content-Encoding") == "" {
			req.Header.Set("Content-Encoding", "gzip")
		}
	}
})

func NewArtifactsCmd() *cobra.Command {
	var (
		mounts            []string
		id                string
		compress          string
		compressCachePath string
		unpack            bool
	)

	cmd := &cobra.Command{
		Use:   "artifacts <paths...>",
		Short: "Save workflow artifacts",
		Args:  cobra.MinimumNArgs(1),

		Run: func(cmd *cobra.Command, paths []string) {
			root, _ := os.Getwd()
			walker, err := artifacts.CreateWalker(paths, mounts, root)
			ui.ExitOnError("building a walker", err)

			if len(walker.Patterns()) == 0 || len(walker.SearchPaths()) == 0 {
				ui.Failf("error: did not found any valid path pattern in the mounted directories")
			}

			fmt.Printf("Root: %s\nPatterns:\n", ui.LightCyan(walker.Root()))
			for _, p := range walker.Patterns() {
				fmt.Printf("- %s\n", ui.LightMagenta(p))
			}
			fmt.Printf("\n")

			// Configure uploader
			var processor artifacts.Processor
			var uploader artifacts.Uploader

			// Sanitize archive name
			compress = strings.Trim(filepath.ToSlash(filepath.Clean(compress)), "/.")
			if compress != "" {
				compressLower := strings.ToLower(compress)
				if strings.HasSuffix(compressLower, ".tar") {
					compress += ".gz"
				} else if !strings.HasSuffix(compressLower, ".tgz") && !strings.HasSuffix(compressLower, ".tar.gz") {
					compress += ".tar.gz"
				}
			}

			// Archive
			if env.CloudEnabled() {
				if compress != "" {
					processor = artifacts.NewTarCachedProcessor(compress, compressCachePath)
					opts := []artifacts.CloudUploaderOpt{cloudAddGzipEncoding}
					if unpack {
						opts = append(opts, cloudUnpack)
					}
					uploader = artifacts.NewCloudUploader(opts...)
				} else {
					processor = artifacts.NewDirectProcessor()
					uploader = artifacts.NewCloudUploader(artifacts.WithParallelismCloud(30), cloudDetectMimetype)
				}
			} else if compress != "" && unpack {
				processor = artifacts.NewTarCachedProcessor(compress, compressCachePath)
				uploader = artifacts.NewDirectUploader(directAddGzipEncoding, directDisableMultipart, directUnpack)
			} else if compress != "" && compressCachePath != "" {
				processor = artifacts.NewTarCachedProcessor(compress, compressCachePath)
				uploader = artifacts.NewDirectUploader(directAddGzipEncoding, directDisableMultipart)
			} else if compress != "" {
				processor = artifacts.NewTarProcessor(compress)
				uploader = artifacts.NewDirectUploader(directAddGzipEncoding)
			} else {
				processor = artifacts.NewDirectProcessor()
				uploader = artifacts.NewDirectUploader(artifacts.WithParallelism(30), directDetectMimetype)
			}

			handler := artifacts.NewHandler(uploader, processor)

			err = handler.Start()
			ui.ExitOnError("initializing uploader", err)

			started := time.Now()
			err = walker.Walk(os.DirFS("/"), func(path string, file fs.File, err error) error {
				if err != nil {
					fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading it: %s\n", path, err.Error())
					return nil
				}

				stat, err := file.Stat()
				if err != nil {
					fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading it: %s\n", path, err.Error())
					return nil
				}
				return handler.Add(path, file, stat)
			})
			ui.ExitOnError("reading the file system", err)
			err = handler.End()

			// TODO: Emit information about artifacts
			ui.ExitOnError("finishing upload", err)
			fmt.Printf("Took %s.\n", time.Now().Sub(started).Truncate(time.Millisecond))
		},
	}

	cmd.Flags().StringSliceVarP(&mounts, "mount", "m", nil, "mounted volumes for limiting paths")
	cmd.Flags().StringVar(&id, "id", "", "execution ID")
	cmd.Flags().StringVar(&compress, "compress", "", "tgz name if should be compressed")
	cmd.Flags().BoolVar(&unpack, "unpack", false, "minio only: unpack the file if compressed")
	cmd.Flags().StringVar(&compressCachePath, "compress-cache", "", "local cache path for passing compressed archive through")

	return cmd
}
