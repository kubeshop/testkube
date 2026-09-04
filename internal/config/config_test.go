package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	assertion := require.New(t)

	cfg, err := Get()
	if err != nil {
		t.Errorf("Get() failed, expected nil, got %v", err)
	}

	assertion.NoError(err)
	assertion.IsType(&Config{}, cfg)
}

// TestExpirationSettingsAreOptIn guards a retention regression that is invisible in
// normal use.
//
// Applying a bucket lifecycle replaces it wholesale, so the API server only touches it
// when an expiration is actually configured. A default on either setting would make
// startup apply Testkube's rules to every installation, dropping the transition and
// expiration rules of anyone whose bucket lifecycle is managed elsewhere - a change in
// how long their objects live, caused by nothing but an upgrade.
func TestExpirationSettingsAreOptIn(t *testing.T) {
	// Unset rather than read whatever the shell happens to hold: this is a test about
	// the declared defaults, so an environment that configures an expiration - as a
	// real deployment does - must not make it fail.
	for _, name := range []string{"STORAGE_EXPIRATION", "STORAGE_CACHE_EXPIRATION"} {
		if previous, ok := os.LookupEnv(name); ok {
			require.NoError(t, os.Unsetenv(name))
			t.Cleanup(func() { _ = os.Setenv(name, previous) })
		}
	}

	cfg, err := Get()
	require.NoError(t, err)

	require.Zero(t, cfg.StorageExpiration,
		"STORAGE_EXPIRATION must stay opt-in: see SetExpirationPolicies")
	require.Zero(t, cfg.StorageCacheExpiration,
		"STORAGE_CACHE_EXPIRATION must stay opt-in: see SetExpirationPolicies")
}
