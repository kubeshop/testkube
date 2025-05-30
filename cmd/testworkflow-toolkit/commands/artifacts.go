package commands

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/filesystem"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/mapper/cdevents"
	"github.com/kubeshop/testkube/pkg/ui"
)

var cloudAddGzipEncoding = artifacts.WithRequestEnhancerCloud(func(req *http.Request, path string, size int64) {
	req.Header.Set("Content-Type", "application/gzip")
	req.Header.Set("Content-Encoding", "gzip")
})

var cloudUnpack = artifacts.WithRequestEnhancerCloud(func(req *http.Request, path string, size int64) {
	req.Header.Set("X-Amz-Meta-Snowball-Auto-Extract", "true")
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

			var handlerOpts []artifacts.HandlerOpts
			// Archive
			cfg := config.Config()
			client, err := env.Cloud()
			if err != nil {
				ui.Failf("could not create cloud client: %v", err)
			}

			if env.HasJunitSupport() {
				junitProcessor := artifacts.NewJUnitPostProcessor(filesystem.NewOSFileSystem(), client, cfg.Execution.EnvironmentId, cfg.Execution.Id, cfg.Workflow.Name, config.Ref(), walker.Root(), cfg.Resource.FsPrefix)
				handlerOpts = append(handlerOpts, artifacts.WithPostProcessor(junitProcessor))
			}
			if compress != "" {
				processor = artifacts.NewTarCachedProcessor(compress, compressCachePath)
				opts := []artifacts.CloudUploaderOpt{cloudAddGzipEncoding}
				if unpack {
					opts = append(opts, cloudUnpack)
				}
				uploader = artifacts.NewCloudUploader(client, cfg.Execution.EnvironmentId, cfg.Execution.Id, cfg.Workflow.Name, config.Ref(), opts...)
			} else {
				processor = artifacts.NewDirectProcessor()
				uploader = artifacts.NewCloudUploader(client, cfg.Execution.EnvironmentId, cfg.Execution.Id, cfg.Workflow.Name, config.Ref(), artifacts.WithParallelismCloud(30), artifacts.CloudDetectMimetype)
			}

			// Isolate the files under specific prefix
			if cfg.Resource.FsPrefix != "" {
				handlerOpts = append(handlerOpts, artifacts.WithPathPrefix(cfg.Resource.FsPrefix))
			}

			// Support cd events
			if cfg.ControlPlane.CDEventsTarget != "" {
				handlerOpts = append(handlerOpts, artifacts.WithCDEventsTarget(cfg.ControlPlane.CDEventsTarget))
				handlerOpts = append(handlerOpts, artifacts.WithCDEventsArtifactParameters(cdevents.CDEventsArtifactParameters{
					Id:           cfg.Execution.Id,
					Name:         cfg.Execution.Name,
					WorkflowName: cfg.Workflow.Name,
					ClusterID:    cfg.Worker.ClusterID,
					DashboardURI: cfg.ControlPlane.DashboardUrl,
				}))
			}

			handler := artifacts.NewHandler(uploader, processor, handlerOpts...)

			run(handler, walker, os.DirFS("/"))
		},
	}

	cmd.Flags().StringSliceVarP(&mounts, "mount", "m", nil, "mounted volumes for limiting paths")
	cmd.Flags().StringVar(&id, "id", "", "execution ID")
	cmd.Flags().StringVar(&compress, "compress", "", "tgz name if should be compressed")
	cmd.Flags().BoolVar(&unpack, "unpack", false, "minio only: unpack the file if compressed")
	cmd.Flags().StringVar(&compressCachePath, "compress-cache", "", "local cache path for passing compressed archive through")

	return cmd
}

func run(handler artifacts.Handler, walker artifacts.Walker, dirFS fs.FS) {
	err := handler.Start()
	ui.ExitOnError("initializing uploader", err)

	started := time.Now()
	err = walker.Walk(dirFS, func(path string, file fs.File, _ fs.FileInfo, err error) error {
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
}
