package volume

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

// The Control Plane ignores an empty restore key on purpose. Scoped it would become
// "<id>/", which every entry on this volume matches as a prefix, so a workflow carrying
// one - from a template that resolved to nothing - would restore whichever entry
// happened to be newest rather than none at all.
func TestScopedKeyLeavesAnEmptyKeyEmpty(t *testing.T) {
	assert.Empty(t, ScopedKey("vol1", ""))
}

// Created rather than linked into place: link(2) is not implemented by every filesystem
// a ReadWriteMany claim can be backed by, and ReadWriteMany describes who may mount a
// volume rather than what the filesystem under it can do. Requiring a link turned the
// cache off on claims that were otherwise perfectly good, and silently.
func TestEnsureIDLeavesNoTemporaryFilesBehind(t *testing.T) {
	root := t.TempDir()

	_, err := EnsureID(root)
	require.NoError(t, err)

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1, "only the identity itself")
	assert.Equal(t, IDName, entries[0].Name())
}

// O_EXCL publishes the name before its contents, so an agent arriving in between finds
// the file empty. Failing there would refuse the volume for that agent - one identity
// for one volume is the whole point, so the loser of the race waits for the winner.
func TestEnsureIDWaitsForAnIdentityBeingWritten(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, IDName)

	// The state another agent's O_EXCL leaves behind before it writes.
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, SharedFileMode)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = os.WriteFile(name, []byte("written-by-the-winner\n"), SharedFileMode)
	}()

	id, err := EnsureID(root)

	require.NoError(t, err, "an identity on its way is not a reason to refuse the volume")
	assert.Equal(t, "written-by-the-winner", id)
}

// An identity that never arrives is reported rather than waited on for ever.
func TestEnsureIDGivesUpOnAnIdentityThatStaysEmpty(t *testing.T) {
	root := t.TempDir()
	f, err := os.OpenFile(filepath.Join(root, IDName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, SharedFileMode)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = EnsureID(root)

	assert.Error(t, err)
}
