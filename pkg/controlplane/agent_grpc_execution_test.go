package controlplane

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func TestListExecutionArtifactsPresigned(t *testing.T) {
	newServer := func(t *testing.T) (*Server, *storage.MockClient) {
		t.Helper()
		storageClient := storage.NewMockClient(gomock.NewController(t))
		return &Server{storageClient: storageClient, cfg: Config{StorageBucket: "artifacts"}}, storageClient
	}

	t.Run("grants a URL per matching artifact", func(t *testing.T) {
		server, storageClient := newServer(t)
		storageClient.EXPECT().ListFilesFromBucket(gomock.Any(), "artifacts", "exec-1").Return([]testkube.Artifact{
			{Name: "results/summary.json", Size: 15},
			{Name: "results/nested/report.xml", Size: 20},
			{Name: "logs/output.txt", Size: 30},
		}, nil)
		storageClient.EXPECT().
			PresignDownloadFileFromBucket(gomock.Any(), "artifacts", "exec-1", "results/summary.json", ArtifactPresignedURLExpiration).
			Return("https://storage/summary", nil)
		storageClient.EXPECT().
			PresignDownloadFileFromBucket(gomock.Any(), "artifacts", "exec-1", "results/nested/report.xml", ArtifactPresignedURLExpiration).
			Return("https://storage/report", nil)

		res, err := server.ListExecutionArtifactsPresigned(context.Background(), &cloud.ListExecutionArtifactsPresignedRequest{
			Id:       "exec-1",
			Patterns: []string{"results/**"},
		})
		require.NoError(t, err)
		require.Len(t, res.Artifacts, 2)
		assert.Equal(t, "results/summary.json", res.Artifacts[0].Path)
		assert.Equal(t, "https://storage/summary", res.Artifacts[0].Url)
		assert.Equal(t, int64(15), res.Artifacts[0].Size)
		assert.Equal(t, "results/nested/report.xml", res.Artifacts[1].Path)
	})

	t.Run("no pattern means every artifact", func(t *testing.T) {
		server, storageClient := newServer(t)
		storageClient.EXPECT().ListFilesFromBucket(gomock.Any(), "artifacts", "exec-1").
			Return([]testkube.Artifact{{Name: "a.txt"}, {Name: "b.txt"}}, nil)
		storageClient.EXPECT().PresignDownloadFileFromBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("https://storage/file", nil).Times(2)

		res, err := server.ListExecutionArtifactsPresigned(context.Background(), &cloud.ListExecutionArtifactsPresignedRequest{Id: "exec-1"})
		require.NoError(t, err)
		assert.Len(t, res.Artifacts, 2)
	})

	t.Run("an execution without artifacts is empty, not an error", func(t *testing.T) {
		server, storageClient := newServer(t)
		storageClient.EXPECT().ListFilesFromBucket(gomock.Any(), "artifacts", "exec-1").
			Return(nil, minio.ErrArtifactsNotFound)

		res, err := server.ListExecutionArtifactsPresigned(context.Background(), &cloud.ListExecutionArtifactsPresignedRequest{Id: "exec-1"})
		require.NoError(t, err)
		assert.Empty(t, res.Artifacts)
	})

	t.Run("requires an execution id", func(t *testing.T) {
		server, _ := newServer(t)
		_, err := server.ListExecutionArtifactsPresigned(context.Background(), &cloud.ListExecutionArtifactsPresignedRequest{})
		assert.ErrorContains(t, err, "execution id is required")
	})
}

func TestMatchesArtifactPatterns(t *testing.T) {
	assert.True(t, matchesArtifactPatterns("results/a.json", nil))
	assert.True(t, matchesArtifactPatterns("results/a.json", []string{"results/*"}))
	assert.True(t, matchesArtifactPatterns("results/nested/a.json", []string{"results/**"}))
	assert.True(t, matchesArtifactPatterns("results/a.json", []string{"logs/*", "results/*"}))
	assert.False(t, matchesArtifactPatterns("results/nested/a.json", []string{"results/*"}))
	assert.False(t, matchesArtifactPatterns("logs/a.txt", []string{"results/**"}))
}
