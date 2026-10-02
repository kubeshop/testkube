package volume

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, name, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o777))
	require.NoError(t, os.WriteFile(name, []byte(body), 0o666))
}

// entryFrom builds a staged entry holding the given declared paths, the way a save
// does, and returns a reader over its mirrored tree.
func entryFrom(t *testing.T, paths []string) *os.Root {
	t.Helper()
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))
	_, _, err := SaveTree(staging, paths, CopyLimits{})
	require.NoError(t, err)
	root, err := os.OpenRoot(staging)
	require.NoError(t, err)
	t.Cleanup(func() { root.Close() })
	return root
}

func TestSaveAndRestoreRoundTripATree(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "pkg", "a.js"), "alpha")
	write(t, filepath.Join(src, "pkg", "nested", "b.js"), "beta")

	entry := entryFrom(t, []string{src})
	dest := t.TempDir()
	require.NoError(t, os.RemoveAll(dest))

	// Restoring writes to the absolute path the entry mirrors, so the destination is
	// the source path itself - which is what a restore into a fresh pod does.
	require.NoError(t, os.RemoveAll(src))
	wrote, err := RestoreTree(entry, []string{src}, CopyLimits{})

	require.NoError(t, err)
	assert.True(t, wrote)
	body, err := os.ReadFile(filepath.Join(src, "pkg", "nested", "b.js"))
	require.NoError(t, err)
	assert.Equal(t, "beta", string(body))
}

// An entry may have been written by another workflow under an environment-scoped
// cache, so anything outside the declared paths is skipped rather than written
// somewhere the step never asked for and the cleanup would not reach.
func TestRestoreSkipsWhatTheStepDidNotDeclare(t *testing.T) {
	declared := posixDir(t)
	smuggled := posixDir(t)
	write(t, filepath.Join(declared, "wanted"), "yes")
	write(t, filepath.Join(smuggled, "unwanted"), "no")

	// The entry carries both, but the restore only declares one.
	entry := entryFrom(t, []string{declared, smuggled})
	require.NoError(t, os.RemoveAll(declared))
	require.NoError(t, os.RemoveAll(smuggled))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(declared, "wanted"))
	assert.NoFileExists(t, filepath.Join(smuggled, "unwanted"), "an undeclared path must not be restored")
}

func TestRestoreStopsAtTheSizeLimit(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "big"), "0123456789")
	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(src))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{MaxTotalBytes: 4})

	assert.ErrorIs(t, err, ErrTooLarge)
}

func TestRestoreStopsAtTheEntryLimit(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "one"), "a")
	write(t, filepath.Join(src, "two"), "b")
	write(t, filepath.Join(src, "three"), "c")
	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(src))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{MaxEntries: 2})

	assert.ErrorIs(t, err, ErrTooManyEntries)
}

// A step may declare a cache path it never creates. That is a smaller entry, not a
// failure - the same way an unmatched hash_files glob is a miss rather than an error.
func TestSaveSkipsAPathThatWasNeverCreated(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "real"), "x")
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, _, err := SaveTree(staging, []string{src, filepath.Join(src, "absent")}, CopyLimits{})

	require.NoError(t, err)
	assert.EqualValues(t, 1, size)
}

func TestSaveReportsTheTotalSize(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "a"), "12345")
	write(t, filepath.Join(src, "b"), "678")
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, _, err := SaveTree(staging, []string{src}, CopyLimits{})

	require.NoError(t, err)
	assert.EqualValues(t, 8, size)
}

// The container root is refused the way the archive form refused it, so a cache cannot
// be declared over the whole filesystem.
func TestSaveRefusesTheRoot(t *testing.T) {
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, _, err := SaveTree(staging, []string{"/"}, CopyLimits{})

	require.NoError(t, err)
	assert.Zero(t, size)
}

// A symlink already sitting inside a declared path is attacker-controlled input: the
// entry may have been written by another workflow under an environment-scoped key. A
// restore that wrote by absolute name would follow it and overwrite a file the step
// never declared, which is what writing through an os.Root on the declared path stops.
func TestRestoreDoesNotFollowASymlinkOutOfTheDeclaredPath(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside")
	require.NoError(t, os.WriteFile(outside, []byte("untouched"), 0o666))

	// The entry carries a plain file at <declared>/escape.
	declared := posixDir(t)
	write(t, filepath.Join(declared, "escape"), "from the entry")
	entry := entryFrom(t, []string{declared})
	require.NoError(t, os.RemoveAll(declared))

	// The destination already holds a symlink at that name, pointing outside.
	require.NoError(t, os.MkdirAll(declared, 0o777))
	require.NoError(t, os.Symlink(outside, filepath.Join(declared, "escape")))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})
	require.NoError(t, err)

	body, readErr := os.ReadFile(outside)
	require.NoError(t, readErr)
	assert.Equal(t, "untouched", string(body),
		"a restore must not write through a symlink that leaves the declared path")
}

// node_modules/.bin is entirely symlinks. Dropping them would restore an incomplete
// tree that still answers as an exact hit, so the step would never repair it.
func TestSaveAndRestorePreserveSymlinks(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "pkg", "cli.js"), "#!/usr/bin/env node")
	require.NoError(t, os.MkdirAll(filepath.Join(src, ".bin"), 0o777))
	require.NoError(t, os.Symlink("../pkg/cli.js", filepath.Join(src, ".bin", "cli")))

	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(src))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{})
	require.NoError(t, err)

	link, readErr := os.Readlink(filepath.Join(src, ".bin", "cli"))
	require.NoError(t, readErr, "the link must come back as a link")
	assert.Equal(t, "../pkg/cli.js", filepath.ToSlash(link),
		"a relative link must stay relative, or it would name the pod that saved it")

	// And it still resolves, which is the whole point of carrying it.
	body, readErr := os.ReadFile(filepath.Join(src, ".bin", "cli"))
	require.NoError(t, readErr)
	assert.Equal(t, "#!/usr/bin/env node", string(body))
}

// Restoring over a tree that is already there is ordinary: a step may declare a path
// its checkout populated.
func TestRestoreOverwritesAnExistingFile(t *testing.T) {
	declared := posixDir(t)
	write(t, filepath.Join(declared, "dep"), "from the entry")
	entry := entryFrom(t, []string{declared})

	require.NoError(t, os.WriteFile(filepath.Join(declared, "dep"), []byte("stale"), 0o666))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})
	require.NoError(t, err)

	body, readErr := os.ReadFile(filepath.Join(declared, "dep"))
	require.NoError(t, readErr)
	assert.Equal(t, "from the entry", string(body))
}

// posixDir returns a temp directory named the way a cached path is: an absolute POSIX
// path. On Windows the drive letter is dropped, which still resolves to the same
// directory on the current drive, so the mirroring these tests exercise behaves as it
// does in the Linux containers the agent actually runs in.
func posixDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.ToSlash(dir[len(filepath.VolumeName(dir)):])
}

// os.Root confines what happens after it is opened; opening it does not confine itself.
// A declared path that is a link to somewhere else would yield a root at that somewhere
// else, and every write - confined, correctly, to the wrong place - would land outside
// the declared path, where the cleanup after a failed restore would not reach it.
func TestRestoreRefusesADeclaredPathThatIsASymlink(t *testing.T) {
	elsewhere := posixDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(elsewhere, "victim"), []byte("untouched"), 0o666))

	// The entry carries <declared>/victim, which would overwrite it if the link were
	// followed.
	declared := posixDir(t)
	write(t, filepath.Join(declared, "victim"), "from the entry")
	entry := entryFrom(t, []string{declared})
	require.NoError(t, os.RemoveAll(declared))
	require.NoError(t, os.Symlink(elsewhere, declared))

	wrote, err := RestoreTree(entry, []string{declared}, CopyLimits{})

	assert.ErrorIs(t, err, ErrDeclaredPathIsSymlink)
	assert.False(t, wrote, "nothing may be written, so the caller must not clear the paths")
	body, readErr := os.ReadFile(filepath.Join(elsewhere, "victim"))
	require.NoError(t, readErr)
	assert.Equal(t, "untouched", string(body))
}

// The same applies to a directory on the way to the declared path: traversing it is how
// the restore would be redirected, whether the link is the last component or not.
func TestRestoreRefusesADeclaredPathReachedThroughASymlink(t *testing.T) {
	elsewhere := posixDir(t)
	require.NoError(t, os.MkdirAll(filepath.Join(elsewhere, "inner"), 0o777))
	require.NoError(t, os.WriteFile(filepath.Join(elsewhere, "inner", "victim"), []byte("untouched"), 0o666))

	parent := posixDir(t)
	declared := parent + "/link/inner"
	write(t, filepath.Join(parent, "link", "inner", "victim"), "from the entry")
	entry := entryFrom(t, []string{declared})
	require.NoError(t, os.RemoveAll(filepath.Join(parent, "link")))
	require.NoError(t, os.Symlink(elsewhere, filepath.Join(parent, "link")))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})

	assert.ErrorIs(t, err, ErrDeclaredPathIsSymlink)
	body, readErr := os.ReadFile(filepath.Join(elsewhere, "inner", "victim"))
	require.NoError(t, readErr)
	assert.Equal(t, "untouched", string(body))
}

// A declared path that does not exist yet is the ordinary case - the step is about to
// create it - and must be made as a real directory rather than refused.
func TestRestoreCreatesAMissingDeclaredPath(t *testing.T) {
	declared := posixDir(t)
	write(t, filepath.Join(declared, "dep"), "from the entry")
	entry := entryFrom(t, []string{declared})
	require.NoError(t, os.RemoveAll(declared))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})

	require.NoError(t, err)
	body, readErr := os.ReadFile(filepath.Join(declared, "dep"))
	require.NoError(t, readErr)
	assert.Equal(t, "from the entry", string(body))
}

// A workflow may legitimately declare both a directory and something beneath it.
// Walking each in turn would store the nested tree twice and count it twice, so the
// volume would refuse a cache on a size or entry limit the archive - whose walker
// crosses the filesystem once - would have accepted.
func TestSaveCountsAnOverlappingPathOnce(t *testing.T) {
	outer := posixDir(t)
	inner := outer + "/packages"
	write(t, filepath.FromSlash(inner+"/dep"), "xx")

	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, entries, err := SaveTree(staging, []string{outer, inner}, CopyLimits{})

	require.NoError(t, err)

	// Compared against declaring it once rather than against a literal, because what
	// is being pinned is that the overlap costs nothing - not how many entries a tree
	// of one file and two directories comes to.
	baseline := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(baseline, 0o777))
	onceSize, onceEntries, err := SaveTree(baseline, []string{outer}, CopyLimits{})
	require.NoError(t, err)

	assert.Equal(t, onceEntries, entries, "one tree declared twice is still one tree")
	assert.EqualValues(t, onceSize, size, "and its bytes must not be counted twice either")
}

// Declaring the nested path first must give the same answer: the cover is about which
// paths contain which, not the order they were written in.
func TestSaveCoversRegardlessOfDeclarationOrder(t *testing.T) {
	outer := posixDir(t)
	inner := outer + "/packages"
	write(t, filepath.FromSlash(inner+"/dep"), "xx")

	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	_, entries, err := SaveTree(staging, []string{inner, outer}, CopyLimits{})
	require.NoError(t, err)

	baseline := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(baseline, 0o777))
	_, onceEntries, err := SaveTree(baseline, []string{outer}, CopyLimits{})
	require.NoError(t, err)

	assert.Equal(t, onceEntries, entries)
}

// A sibling whose name merely starts with another's is not inside it, so dropping it
// would silently stop caching a declared path.
func TestCoverPathsKeepsASiblingWithASharedPrefix(t *testing.T) {
	assert.Equal(t,
		[]string{"/data/deps", "/data/deps2"},
		coverPaths([]string{"/data/deps2", "/data/deps"}))
}

func TestCoverPathsDropsWhatAnotherPathContains(t *testing.T) {
	assert.Equal(t,
		[]string{"/a", "/b"},
		coverPaths([]string{"/a", "/a/b", "/a/b/c", "/b", "/a"}))
}

// The root, empty strings and unclean spellings are the shapes that would otherwise
// cover everything else and cache the whole filesystem.
func TestCoverPathsDiscardsUnusablePaths(t *testing.T) {
	assert.Empty(t, coverPaths([]string{"", "/", ".", "/.."}))
	assert.Equal(t, []string{"/data/deps"}, coverPaths([]string{"/data/./deps", "/data/deps/"}))
}

// The restore side covers too, so a nested declared path is not copied out twice.
func TestRestoreCoversOverlappingPaths(t *testing.T) {
	outer := posixDir(t)
	inner := outer + "/packages"
	write(t, filepath.FromSlash(inner+"/dep"), "cached")
	entry := entryFrom(t, []string{outer})
	require.NoError(t, os.RemoveAll(filepath.FromSlash(outer)))

	// Two: the nested directory and the file in it. Declaring the nested path as well
	// must not double that.
	_, err := RestoreTree(entry, []string{outer, inner}, CopyLimits{MaxEntries: 2})

	require.NoError(t, err, "the nested path must not be counted a second time")
	body, readErr := os.ReadFile(filepath.FromSlash(inner + "/dep"))
	require.NoError(t, readErr)
	assert.Equal(t, "cached", string(body))
}

// wrote is what tells the caller to clear the declared paths after a failed restore, so
// it has to mean "this touched the filesystem", not "this copied bytes".
//
// A restore that makes a directory tree and an empty file and then stops at the entry
// limit has changed the destination. Reporting nothing written leaves that behind while
// telling the step it was a plain miss, so the install runs over a half-restored tree
// nothing will clean up.
func TestRestoreReportsADirectoryAndAnEmptyFileAsWritten(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.FromSlash(src+"/nested/a"), "")
	write(t, filepath.FromSlash(src+"/nested/b"), "second")
	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(filepath.FromSlash(src)))

	// Two entries are allowed - the directory is one of them - so the empty file is
	// restored and the next one is refused. Nothing with any bytes in it is ever copied.
	wrote, err := RestoreTree(entry, []string{src}, CopyLimits{MaxEntries: 2})

	require.ErrorIs(t, err, ErrTooManyEntries)
	assert.True(t, wrote, "the directory and the empty file are both left behind")

	_, statErr := os.Stat(filepath.FromSlash(src + "/nested/a"))
	require.NoError(t, statErr, "and really were created, so the test is pinning the right thing")
}

// A path the entry does not carry is a normal thing to find - an entry smaller than
// the step declared still restores. Anything else the volume says is not that.
//
// Reading an I/O or permission error as "absent" would return a successful restore of
// nothing. The caller reports that as an exact hit, so the save stage skips replacing
// the entry, and where one declared path restored and another failed this way the tree
// looks whole and is not.
func TestRestoreReportsAVolumeFailureRatherThanAnAbsentPath(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.FromSlash(src+"/dep"), "cached")
	entry := entryFrom(t, []string{src})

	// Standing in for the volume going away mid-restore, which is what an unreadable
	// entry looks like from here - and unlike a permission bit, it reads the same on
	// every platform.
	require.NoError(t, entry.Close())

	wrote, err := RestoreTree(entry, []string{src}, CopyLimits{})

	require.Error(t, err, "a volume that cannot be read is not an entry without this path")
	assert.NotErrorIs(t, err, fs.ErrNotExist)
	assert.False(t, wrote)
}

// A declared path is relative whenever the step's own image decides the working
// directory, which is the common case - mountCachePaths resolves one against the
// working directory only where the bundle already knows it.
//
// The destination walk used to start at the filesystem root regardless, so it created
// /node_modules, left ./node_modules alone, and os.OpenRoot then failed on a directory
// nothing had made: the volume never restored a relative path at all. It also ran the
// symlink guard against a path other than the one about to be written to.
func TestRestoreCreatesARelativeDeclaredPathWhereTheStepLooksForIt(t *testing.T) {
	t.Chdir(t.TempDir())
	write(t, filepath.Join("node_modules", "dep"), "cached")

	// Built from the same spelling the step declared, as a save does.
	entry := entryFrom(t, []string{"node_modules"})
	require.NoError(t, os.RemoveAll("node_modules"))

	wrote, err := RestoreTree(entry, []string{"node_modules"}, CopyLimits{})

	require.NoError(t, err)
	assert.True(t, wrote)
	body, readErr := os.ReadFile(filepath.Join("node_modules", "dep"))
	require.NoError(t, readErr, "the restore must land where the step will look")
	assert.Equal(t, "cached", string(body))
}

// A key says nothing about the paths it was saved with - it is whatever the workflow
// templated, commonly a lockfile hash - so an environment-scoped key shared by
// workflows that cache different directories, or a workflow that changes its paths
// without changing its key, reaches an entry holding none of what it asked for.
//
// Reporting success there records an exact hit that restored nothing, and an exact hit
// tells the save stage there is nothing to replace: the step would reinstall on every
// execution while the entry went on claiming to hold what it does not.
func TestRestoreReportsAnEntryHoldingNoneOfTheDeclaredPaths(t *testing.T) {
	stored := posixDir(t)
	write(t, filepath.FromSlash(stored+"/dep"), "cached")
	entry := entryFrom(t, []string{stored})

	// What this step asks for was never in the entry.
	asked := posixDir(t)

	wrote, err := RestoreTree(entry, []string{asked}, CopyLimits{})

	require.ErrorIs(t, err, ErrEntryHoldsNoDeclaredPath)
	assert.False(t, wrote, "nothing was written, so there is nothing to clear up")
}

// An entry smaller than the step declared is the ordinary shape of one: SaveTree skips
// a declared path that did not exist when the entry was written. Some is still a hit.
func TestRestoreAcceptsAnEntryHoldingOnlySomeDeclaredPaths(t *testing.T) {
	present := posixDir(t)
	write(t, filepath.FromSlash(present+"/dep"), "cached")
	entry := entryFrom(t, []string{present})
	require.NoError(t, os.RemoveAll(filepath.FromSlash(present)))
	absent := posixDir(t) + "/never-saved"

	wrote, err := RestoreTree(entry, []string{present, absent}, CopyLimits{})

	require.NoError(t, err)
	assert.True(t, wrote)
	body, readErr := os.ReadFile(filepath.FromSlash(present + "/dep"))
	require.NoError(t, readErr)
	assert.Equal(t, "cached", string(body))
}

// A directory costs an inode on the shared volume and a mkdir on every restore, and
// nothing else bounds one: MaxTotalBytes weighs file contents, of which a directory has
// none. A step controls what sits under its own cached paths, so a tree of empty
// directories would otherwise be copied onto a volume every execution shares.
func TestSaveCountsDirectories(t *testing.T) {
	src := posixDir(t)
	require.NoError(t, os.MkdirAll(filepath.FromSlash(src+"/a/b/c"), 0o777))

	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	_, _, err := SaveTree(staging, []string{src}, CopyLimits{MaxEntries: 2})

	assert.ErrorIs(t, err, ErrTooManyEntries, "four directories and no files is still four entries")
}

// Counted on both sides, or a tree passes the limit going in and fails it coming out -
// an entry that stores and is then refused by every restore of it, under a key no later
// run can replace.
func TestRestoreCountsDirectories(t *testing.T) {
	src := posixDir(t)
	require.NoError(t, os.MkdirAll(filepath.FromSlash(src+"/a/b/c"), 0o777))
	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(filepath.FromSlash(src)))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{MaxEntries: 2})

	assert.ErrorIs(t, err, ErrTooManyEntries)
}

// Counting directories against the limit must not make a tree of nothing but
// directories look like something worth storing. It restores no files, so publishing it
// under an immutable key would answer every later run with a hit holding nothing - the
// case the caller's emptiness check exists for.
func TestSaveReportsNoContentForADirectoryOnlyTree(t *testing.T) {
	src := posixDir(t)
	require.NoError(t, os.MkdirAll(filepath.FromSlash(src+"/a/b"), 0o777))

	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	_, content, err := SaveTree(staging, []string{src}, CopyLimits{})

	require.NoError(t, err)
	assert.Zero(t, content, "directories are work, not content")
}

// A declared path that does not exist has to leave nothing at all in the entry.
//
// Creating its mirrored prefix up front put an empty tree there, and a restore reads
// the presence of that directory as the path being carried: it reports an exact hit
// having written nothing, and an exact hit tells the save stage there is nothing to
// replace. The step would then reinstall on every execution while the entry went on
// claiming to hold a path it never held.
func TestSaveLeavesNothingForAPathThatDoesNotExist(t *testing.T) {
	present := posixDir(t)
	write(t, filepath.FromSlash(present+"/dep"), "cached")
	absent := posixDir(t) + "/never-created"

	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))
	_, _, err := SaveTree(staging, []string{present, absent}, CopyLimits{})
	require.NoError(t, err)

	mirrored := filepath.Join(staging, filepath.FromSlash(strings.TrimPrefix(absent, "/")))
	_, statErr := os.Stat(mirrored)
	assert.True(t, os.IsNotExist(statErr), "an absent path must not be mirrored into the entry")

	// And the entry is therefore honest about what it holds: a restore asking only for
	// the absent path gets a miss rather than a hit over an empty directory.
	entry, openErr := os.OpenRoot(staging)
	require.NoError(t, openErr)
	defer entry.Close()

	_, restoreErr := RestoreTree(entry, []string{absent}, CopyLimits{})
	assert.ErrorIs(t, restoreErr, ErrEntryHoldsNoDeclaredPath)
}

// copyOut has to say whether it staged anything, because its error alone cannot: a
// source that vanished between the walk listing it and the open is skipped with a nil
// error, since an entry is a snapshot rather than a transaction.
//
// Counting it anyway publishes an entry of nothing but the directories leading to it as
// though it held content, and every later restore then reports an exact hit having
// written nothing, under a key no run can replace.
func TestCopyOutReportsNothingStagedWhenTheSourceVanished(t *testing.T) {
	source := filepath.Join(t.TempDir(), "dep")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o666))
	info, err := os.Lstat(source)
	require.NoError(t, err)

	// Listed by the walk, gone by the time it is opened.
	require.NoError(t, os.Remove(source))

	target := filepath.Join(t.TempDir(), "staged")
	n, staged, copyErr := copyOut(source, target, info, 1<<20)

	require.NoError(t, copyErr, "a vanished source is skipped, not failed")
	assert.False(t, staged, "and must not be counted as content")
	assert.Zero(t, n)
	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr), "nothing was written")
}

// The counterpart: a source that is there is staged and says so.
func TestCopyOutReportsStagedWhenItWrites(t *testing.T) {
	source := filepath.Join(t.TempDir(), "dep")
	require.NoError(t, os.WriteFile(source, []byte("installed"), 0o666))
	info, err := os.Lstat(source)
	require.NoError(t, err)

	target := filepath.Join(t.TempDir(), "staged")
	n, staged, copyErr := copyOut(source, target, info, 1<<20)

	require.NoError(t, copyErr)
	assert.True(t, staged)
	assert.EqualValues(t, len("installed"), n)
}
