package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	storageMinio "github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

type MinioOptions struct {
	Path      string
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Secure    bool
	Region    string
}

func NewMinioCmd() *cobra.Command {
	opts := &MinioOptions{}

	cmd := &cobra.Command{
		Use:   "minio <endpoint> <outputPath>",
		Short: "Download test content from MinIO/S3 using Testkube’s MinIO client",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunMinio(cmd.Context(), args[0], args[1], opts); err != nil {
				ui.Fail(err)
			}
		},
	}

	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "MinIO/S3 path to download from")
	cmd.Flags().StringVarP(&opts.Bucket, "bucket", "b", "", "MinIO/S3 bucket name")
	cmd.Flags().StringVarP(&opts.AccessKey, "accessKey", "a", "", "MinIO/S3 access key")
	cmd.Flags().StringVarP(&opts.SecretKey, "secretKey", "s", "", "MinIO/S3 secret key")
	cmd.Flags().StringVarP(&opts.Region, "region", "r", "", "MinIO/S3 region")
	return cmd
}

func RunMinio(ctx context.Context, endpoint, outputPath string, opts *MinioOptions) error {
	// Normalize destination
	destinationPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destinationPath, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	fmt.Printf("📦 Downloading from MinIO bucket from endpoint %s, bucket %s, path %s...\n",
		endpoint, opts.Bucket, opts.Path)

	// Create MinIO client
	minioClient := storageMinio.NewClient(
		endpoint,
		opts.AccessKey,
		opts.SecretKey,
		opts.Region,
		"",
		opts.Bucket,
	)

	// Connect
	if err := minioClient.Connect(); err != nil {
		return fmt.Errorf("connecting to MinIO: %w", err)
	}

	var bucketFolders []string
	cleanPath := strings.Trim(opts.Path, "/")

	if cleanPath == "" || cleanPath == "." {
		bucketFolders = []string{""}
	} else {
		bucketFolders = []string{cleanPath + "/"}
	}

	if err := minioClient.PlaceFiles(ctx, bucketFolders, destinationPath); err != nil {
		return fmt.Errorf("downloading files: %w", err)
	}

	if err := adjustFilePermissions(destinationPath); err != nil {
		return fmt.Errorf("adjusting permissions: %w", err)
	}

	if err := listDirectoryContents(destinationPath); err != nil {
		return fmt.Errorf("listing directory contents: %w", err)
	}

	fmt.Printf("✅ Successfully downloaded MinIO content to %s\n", destinationPath)
	return nil
}
