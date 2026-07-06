package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestNewMetricsRecorderConfig_SkipFlag(t *testing.T) {
	t.Run("Skip is false when DisableResourceMetrics is false and skip arg is false", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = false

		cfg := newMetricsRecorderConfig("step1", false, testworkflowconfig.ContainerResourceConfig{})

		assert.False(t, cfg.Skip)
	})

	t.Run("Skip is true when DisableResourceMetrics is true regardless of skip arg", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = true

		cfg := newMetricsRecorderConfig("step1", false, testworkflowconfig.ContainerResourceConfig{})

		assert.True(t, cfg.Skip)
	})

	t.Run("Skip is true when skip arg is true regardless of DisableResourceMetrics", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = false

		cfg := newMetricsRecorderConfig("step1", true, testworkflowconfig.ContainerResourceConfig{})

		assert.True(t, cfg.Skip)
	})

	t.Run("Skip is true when both DisableResourceMetrics and skip arg are true", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = true

		cfg := newMetricsRecorderConfig("step1", true, testworkflowconfig.ContainerResourceConfig{})

		assert.True(t, cfg.Skip)
	})
}

type fakeArtifactStorage struct {
	saved   []string
	failFor map[string]bool
}

func (s *fakeArtifactStorage) FullPath(artifactPath string) string { return artifactPath }
func (s *fakeArtifactStorage) SaveStream(artifactPath string, stream io.Reader) error {
	return nil
}
func (s *fakeArtifactStorage) SaveFile(artifactPath string, r io.Reader, info os.FileInfo) error {
	if s.failFor[filepath.Base(artifactPath)] {
		return fmt.Errorf("save failed")
	}
	s.saved = append(s.saved, artifactPath)
	return nil
}
func (s *fakeArtifactStorage) Wait() error { return nil }

func TestCollectMetricsFiles(t *testing.T) {
	tests := []struct {
		name      string
		dir       func(t *testing.T) string
		wantFiles int
		wantErr   bool
	}{
		{
			name:      "missing directory means nothing to record",
			dir:       func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") },
			wantFiles: 0,
		},
		{
			name:      "empty directory yields no files",
			dir:       func(t *testing.T) string { return t.TempDir() },
			wantFiles: 0,
		},
		{
			name: "collects files and skips directories",
			dir: func(t *testing.T) string {
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}"), 0644))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.json"), []byte("{}"), 0644))
				return dir
			},
			wantFiles: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := collectMetricsFiles(tt.dir(t))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, files, tt.wantFiles)
		})
	}
}

func TestUploadMetricsFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.json"), []byte("{}"), 0644))
	files, err := collectMetricsFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	t.Run("uploads all files", func(t *testing.T) {
		storage := &fakeArtifactStorage{}
		require.NoError(t, uploadMetricsFiles(storage, files))
		assert.Len(t, storage.saved, 2)
	})

	t.Run("one failure does not stop the remaining uploads", func(t *testing.T) {
		storage := &fakeArtifactStorage{failFor: map[string]bool{"a.json": true}}
		err := uploadMetricsFiles(storage, files)
		require.Error(t, err)
		assert.Len(t, storage.saved, 1)
	})

	t.Run("unreadable file is reported but others are saved", func(t *testing.T) {
		missing := append([]metricsFile{{path: filepath.Join(dir, "gone.json"), info: files[0].info}}, files...)
		storage := &fakeArtifactStorage{}
		err := uploadMetricsFiles(storage, missing)
		require.Error(t, err)
		assert.Len(t, storage.saved, 2)
	})
}
