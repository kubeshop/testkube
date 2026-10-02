package volume

import (
	"os"
	"path/filepath"
	"sync"
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

// The name is published by O_EXCL before the contents are written. An agent killed in
// that window - an eviction, or a node going away between two syscalls - leaves an empty
// identity nobody will ever write, and nothing afterwards repaired it: every later
// startup waited out the attempts, found it still empty and refused the volume, turning
// the cache off for the whole installation permanently, over six bytes.
func TestEnsureIDRecoversAnIdentityNobodyWillWrite(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, IDName)

	require.NoError(t, os.WriteFile(name, nil, SharedFileMode))
	stale := time.Now().Add(-2 * idStaleAfter)
	require.NoError(t, os.Chtimes(name, stale, stale))

	id, err := EnsureID(root)
	require.NoError(t, err)
	assert.NotEmpty(t, id, "an identity left unwritten has to be replaceable")

	// And the replacement is what the volume now carries, so every agent agrees on it.
	again, err := EnsureID(root)
	require.NoError(t, err)
	assert.Equal(t, id, again)

	// The abandoned name is left exactly as it was found. Nothing here unlinks, which
	// is what keeps concurrent agents from removing each other's work, so the volume
	// carries the dead name rather than reusing it.
	info, statErr := os.Stat(name)
	require.NoError(t, statErr)
	assert.Zero(t, info.Size(), "the abandoned identity must not be written or removed")
}

// The same empty file is what another agent in the middle of its own create leaves
// behind, one write wide. Taking that for a stale one would hand the two agents separate
// identities for one volume, splitting its cache between them - so only age tells them
// apart, and a fresh one is waited on exactly as before.
func TestEnsureIDLeavesAnIdentityAnotherAgentIsWriting(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, IDName)
	require.NoError(t, os.WriteFile(name, nil, SharedFileMode))

	_, err := EnsureID(root)
	require.Error(t, err, "an empty identity written a moment ago is somebody's open write")

	_, statErr := os.Stat(name)
	assert.NoError(t, statErr, "and it must still be there for them to finish")
}

// Agents superseding an unwritten identity together must all end up with the same one.
//
// They are not phased apart by chance: each reaches this having waited out the same
// attempts, so they arrive within microseconds of each other. Any repair that unlinks -
// the abandoned identity, or a lock taken to guard it - lets one agent remove what
// another has just created and written, leaving each with an identity the other has
// never seen; and since the identity prefixes every cache key, the volume's cache is
// then split between them until both restart, each missing on every entry the other
// saved. An O_EXCL create on a name that is never reused has no such window.
func TestConcurrentRecoveriesAgreeOnOneIdentity(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, IDName)

	require.NoError(t, os.WriteFile(name, nil, SharedFileMode))
	stale := time.Now().Add(-2 * idStaleAfter)
	require.NoError(t, os.Chtimes(name, stale, stale))

	const agents = 8
	ids := make([]string, agents)
	errs := make([]error, agents)
	var wg sync.WaitGroup
	for i := 0; i < agents; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ids[i], errs[i] = EnsureID(root)
		}(i)
	}
	wg.Wait()

	for i := range ids {
		require.NoError(t, errs[i], "agent %d", i)
		assert.Equal(t, ids[0], ids[i], "agent %d repaired its way to a different identity", i)
	}

	// And they all agreed on one generation rather than each taking its own: with
	// eight agents and no reuse of a name, a volume that ends up carrying more than one
	// new identity is one where the election did not happen.
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "the abandoned identity and exactly one successor")
}

// A superseded name is never unlinked, so it can still be written - by the very agent
// whose stall abandoned it, waking long after everyone else moved on. Reading the names
// in order and taking the first that holds an identity would hand the volume back to it,
// and every agent already running would keep using the successor: one volume, two key
// prefixes, lastingly. The later generation wins instead.
func TestALaterGenerationWinsOverAnAbandonedNameWrittenLate(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, IDName)

	require.NoError(t, os.WriteFile(name, nil, SharedFileMode))
	stale := time.Now().Add(-2 * idStaleAfter)
	require.NoError(t, os.Chtimes(name, stale, stale))

	id, err := EnsureID(root)
	require.NoError(t, err)

	// The stalled agent finally gets its write in, into the name it created.
	require.NoError(t, os.WriteFile(name, []byte("0123456789ab\n"), SharedFileMode))

	again, err := EnsureID(root)
	require.NoError(t, err)
	assert.Equal(t, id, again, "the volume changed identity under the agents using it")
}
