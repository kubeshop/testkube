package featureflags

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	os.Setenv("FF_LOGS_V2", "true")
	t.Cleanup(func() {
		os.Unsetenv("FF_LOGS_V2")
	})

	assertion := require.New(t)

	cfg, err := Get()
	if err != nil {
		t.Errorf("Get() failed, expected nil, got %v", err)
	}

	assertion.NoError(err)
	assertion.IsType(FeatureFlags{}, cfg)
	assertion.True(cfg.LogsV2)
}
