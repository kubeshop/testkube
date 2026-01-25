package artifacts

import (
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
)

// InternalStorageWithConfig creates storage with explicit configuration and client
func InternalStorageWithConfig(cfg *config.ConfigV2, client controlplaneclient.Client) (InternalArtifactStorage, error) {
	uploader := NewCloudUploader(
		client,
		cfg.Internal().Execution.EnvironmentId,
		cfg.Internal().Execution.Id,
		cfg.Internal().Workflow.Name,
		cfg.Ref(),
		WithParallelismCloud(30),
		CloudDetectMimetype,
	)

	return &internalArtifactStorageV2{
		prefix:   ".testkube/" + cfg.Ref(),
		uploader: uploader,
	}, nil
}

// InternalStorageWithProvider creates storage with a custom provider
func InternalStorageWithProvider(provider StorageProvider, cfg *config.ConfigV2) (InternalArtifactStorage, error) {
	uploader, err := provider.GetUploader(
		cfg.Internal().Execution.EnvironmentId,
		cfg.Internal().Execution.Id,
		cfg.Internal().Workflow.Name,
		cfg.Ref(),
	)
	if err != nil {
		return nil, err
	}

	return &internalArtifactStorageV2{
		prefix:   ".testkube/" + cfg.Ref(),
		uploader: uploader,
	}, nil
}
