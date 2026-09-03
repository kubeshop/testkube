package config

import (
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
	cfg, err := Get()
	require.NoError(t, err)

	require.Zero(t, cfg.StorageExpiration,
		"STORAGE_EXPIRATION must stay opt-in: see SetExpirationPolicies")
	require.Zero(t, cfg.StorageCacheExpiration,
		"STORAGE_CACHE_EXPIRATION must stay opt-in: see SetExpirationPolicies")
}
