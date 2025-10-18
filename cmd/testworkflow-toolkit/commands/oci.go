package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/pkg/archive"
	"github.com/regclient/regclient/types/descriptor"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/spf13/cobra"
)

const (
	// OciRetryOnFailureMaxAttempts defines maximum retry attempts for OCI operations
	OciRetryOnFailureMaxAttempts = 5
	// OciRetryOnFailureBaseDelay defines base delay between retries
	OciRetryOnFailureBaseDelay = 100 * time.Millisecond
)

// OciOptions encapsulates all options for the OCI command
type OciOptions struct {
	Path      string
	MountPath string
	Username  string
	Token     string
	Registry  string
}

// NewOciCmd creates a new OCI command
func NewOciCmd() *cobra.Command {
	opts := &OciOptions{}

	cmd := &cobra.Command{
		Use:   "oci <image> <output-path>",
		Short: "Perform OCI operations",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunOci(cmd.Context(), args[0], args[1], opts); err != nil {
				ui.Fail(err)
			}
		},
	}

	cmd.Flags().StringVarP(&opts.Path, "path", "p", ".", "path to extract the artifact content from (relative to artifact root)")
	cmd.Flags().StringVarP(&opts.MountPath, "mountPath", "m", "", "where to mount the fetched content (defaults to \"repo\" directory in the data volume)")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "oci registry username")
	cmd.Flags().StringVarP(&opts.Token, "token", "t", "", "oci registry token")
	cmd.Flags().StringVarP(&opts.Registry, "registry", "r", "", "oci registry")

	return cmd
}

func RunOci(ctx context.Context, image, outputPath string, opts *OciOptions) error {
	fmt.Printf("Starting OCI fetch for artifact: %s to path: %s\n", image, outputPath)
	destinationPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Use temporary directory for cloning
	tmpPath, err := os.MkdirTemp(constants.DefaultTmpDirPath, "oci-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory: %w", err)
	}
	// Ensure cleanup on any error
	defer func() {
		if err = os.RemoveAll(tmpPath); err != nil {
			err = fmt.Errorf("error cleaning up temporary directory: %w", err)
		}
	}()

	// Perform OCI get operation
	if err := performOCIGet(ctx, image, tmpPath, opts); err != nil {
		return fmt.Errorf("error getting OCI image: %w", err)
	}

	// Handle extraction of the artifact content
	if err := handleArtifactExtraction(tmpPath, destinationPath, opts.Path); err != nil {
		return err
	}

	// Adjust file permissions
	if err := adjustFilePermissions(destinationPath); err != nil {
		return fmt.Errorf("error adjusting permissions: %w", err)
	}

	// List final contents
	if err := listDirectoryContents(destinationPath); err != nil {
		fmt.Printf("⚠️ error listing directory contents: %s\n", err.Error())
	}

	fmt.Printf("✅ Successfully fetched OCI artifact: %s\n", image)
	return nil
}

func performOCIGet(ctx context.Context, image, tmpPath string, opts *OciOptions) error {
	// Set up authentication if provided
	var rcOpts []regclient.Opt

	if opts.Username != "" && opts.Token != "" {
		hostConfig := config.Host{
			Name: opts.Registry,
			User: opts.Username,
			Pass: opts.Token,
		}
		rcOpts = append(rcOpts, regclient.WithConfigHost(hostConfig))
	}

	// Create regclient (following the example exactly)
	rc := regclient.New(rcOpts...)

	// Create image reference
	r, err := ref.New(image)
	if err != nil {
		return fmt.Errorf("failed to create image reference: %w", err)
	}

	fmt.Printf("Downloading and extracting artifact...\n")
	m, err := rc.ManifestGet(ctx, r)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	mi, ok := m.(manifest.Imager)
	if !ok {
		return fmt.Errorf("manifest is not an Imager type")
	}
	layers, err := mi.GetLayers()
	if err != nil {
		return fmt.Errorf("failed to get layers: %w", err)
	}

	for _, layer := range layers {
		if err := extractLayerToDir(ctx, rc, r, layer, tmpPath); err != nil {
			return fmt.Errorf("failed to extract layer: %w", err)
		}
	}

	fmt.Printf("✅ OCI artifact fetched successfully.\n")
	return nil
}

func extractLayerToDir(ctx context.Context, rc *regclient.RegClient, r ref.Ref, layer descriptor.Descriptor, outputDir string) error {
	// Get the layer blob
	blobReader, err := rc.BlobGet(ctx, r, layer)
	if err != nil {
		return fmt.Errorf("failed to get blob: %w", err)
	}
	defer blobReader.Close()

	return archive.Extract(ctx, outputDir, blobReader)
}

func handleArtifactExtraction(src, dest, path string) error {
	sourcePath := src
	if path != "" {
		sourcePath = filepath.Join(src, path)
	}

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		if path != "" {
			return fmt.Errorf("artifact path not found: %s", path)
		}
		// If no specific path and nothing was extracted, this might be an error
		if entries, _ := os.ReadDir(src); len(entries) == 0 {
			return fmt.Errorf("no content found in OCI artifact")
		}
	}

	// Copy artifact contents to destination
	fmt.Printf("Moving artifact contents to %s...\n", dest)
	return copyDirContents(sourcePath, dest)
}
