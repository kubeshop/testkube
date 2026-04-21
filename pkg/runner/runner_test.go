package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/internal/config"
)

func TestRunnerLogArchiveEnabledUsesControlPlaneStorageSupport(t *testing.T) {
	t.Run("enabled when control plane advertises log storage support", func(t *testing.T) {
		r := &runner{
			proContext: config.ProContext{
				CloudStorage:                        false,
				CloudStorageSupportedInControlPlane: true,
			},
		}

		require.True(t, r.logArchiveEnabled())
	})

	t.Run("disabled when control plane does not advertise log storage support", func(t *testing.T) {
		r := &runner{
			proContext: config.ProContext{
				CloudStorage:                        true,
				CloudStorageSupportedInControlPlane: false,
			},
		}

		require.False(t, r.logArchiveEnabled())
	})
}
