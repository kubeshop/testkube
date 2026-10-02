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

// TestExpirationDefaults pins the asymmetry between the two expiration settings, which
// is easy to flatten by accident and expensive to get wrong in either direction.
//
// STORAGE_CACHE_EXPIRATION defaults to a day. A cache entry is keyed on the contents of
// a lockfile, so it is disposable: one still wanted is rewritten by the next run that
// misses it. Its lifecycle rule is confined to the cache prefix, so it can only ever
// delete caches.
//
// STORAGE_EXPIRATION stays opt-in. That rule is unfiltered and governs every object in
// the bucket - artifacts and logs included - so a default there would start deleting a
// deployment's results on an upgrade. The difference between the two is the filter, not
// taste.
func TestExpirationDefaults(t *testing.T) {
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
		"STORAGE_EXPIRATION must stay opt-in: its rule is unfiltered and would expire artifacts too")
	require.Equal(t, 1, cfg.StorageCacheExpiration,
		"STORAGE_CACHE_EXPIRATION should default to a day, the shortest an object store lifecycle can express")
}
