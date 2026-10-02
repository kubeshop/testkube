package volume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The identity belongs to the volume, so it has to survive restarts: a new one would
// split the volume's cache in half, and every key stored under the old one would go
// cold until it expired.
func TestEnsureIDIsStableAcrossCalls(t *testing.T) {
	root := t.TempDir()

	first, err := EnsureID(root)
	require.NoError(t, err)
	require.NotEmpty(t, first)

	second, err := EnsureID(root)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

// Two volumes must not answer to one name, which is the whole point of it.
func TestEnsureIDDiffersPerVolume(t *testing.T) {
	a, err := EnsureID(t.TempDir())
	require.NoError(t, err)
	b, err := EnsureID(t.TempDir())
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
}

// Nothing to identify, and nothing to partition - the object store is shared anyway.
func TestEnsureIDIsEmptyWithoutAVolume(t *testing.T) {
	id, err := EnsureID("")

	require.NoError(t, err)
	assert.Empty(t, id)
}

func TestEnsureIDFailsWhenTheVolumeCannotBeRead(t *testing.T) {
	root := t.TempDir()
	// A directory where the identity file belongs: readable, and not an identity.
	require.NoError(t, os.Mkdir(filepath.Join(root, IDName), 0o777))

	_, err := EnsureID(root)

	assert.Error(t, err)
}

// A prefix, not a suffix: a restore key is matched as a prefix, so "npm-" suffixed
// would still match another volume's "npm-abc" - an entry this runner cannot read.
func TestScopedKeyPrefixesAndIsInertWithoutAnID(t *testing.T) {
	assert.Equal(t, "vol1/npm-abc", ScopedKey("vol1", "npm-abc"))
	assert.Equal(t, "npm-abc", ScopedKey("", "npm-abc"))
}
