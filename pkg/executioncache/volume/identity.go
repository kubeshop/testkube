package volume

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// IDName holds the volume's identity, at its root.
	IDName = ".volume-id"

	// EnvVolumeID carries that identity to the pod, which cannot read the volume root
	// on the save side - its only writable mount is a subPath - and should not have to
	// on the restore side either.
	EnvVolumeID = "TK_CACHE_VOLUME_ID"

	// idBytes is how much randomness the identity carries. Short on purpose: it is
	// spent out of MaxKeyBytes, which bounds a user's own key.
	idBytes = 6

	// idReadAttempts and idReadDelay wait out the window where another agent has
	// created the identity and not yet written it. One write wide, and once in a
	// volume's life - long enough to survive a slow network filesystem, short enough
	// that a file which is genuinely empty is not waited on for long.
	idReadAttempts = 20
	idReadDelay    = 50 * time.Millisecond

	// idRecoveryName elects a single agent to repair an identity that was created and
	// never written. Beside it at the volume root, which only agents can reach.
	idRecoveryName = IDName + ".recovering"

	// idStaleAfter is how long an empty identity has to have sat there before it is
	// taken to be one nobody is going to write.
	//
	// Far longer than the write it is waiting on, because the mtime was set by whichever
	// node created the file and is read here by another: minutes of clock skew between
	// two nodes of one cluster is ordinary, and the cost of being wrong is a slow writer
	// ending up with an identity no other agent shares. An agent that stalled five
	// minutes between creating the name and writing six bytes has worse problems.
	idStaleAfter = 5 * time.Minute
)

// errIDRecoveryHeld reports that another agent is repairing the identity. Its write is
// what this agent is waiting for, so the answer is to read again rather than to fail.
var errIDRecoveryHeld = errors.New("another agent is repairing the volume identity")

// EnsureID reads the volume's identity, writing one the first time.
//
// A cache key is shared by every runner in an environment, but an entry on a volume is
// reachable only from that volume: executions choose a runner through spec.target, and
// the scope a key resolves to carries no runner or volume. Without this, a runner that
// saved first would own the key for its whole lifetime, and every runner on a different
// volume would get an exact hit it cannot follow - and could not publish a replacement,
// because the object is immutable. In a two-runner environment that is about half of
// all runs never caching anything, and quietly, because an unfollowable pointer
// degrades to a plain miss.
//
// Prefixing the key with this partitions the namespace instead: runners on different
// volumes simply never ask for the same key.
//
// The identity belongs to the volume rather than to the runner, so several runners that
// do share one volume go on sharing its entries, which is the arrangement worth having.
func EnsureID(mountPath string) (string, error) {
	if mountPath == "" {
		return "", nil
	}
	name := filepath.Join(mountPath, IDName)

	// Twice at most. The second pass is for having found another agent already
	// repairing the identity: its write is the one being waited for, so the way to get
	// the identity is to go round and read it. A third pass would say nothing new.
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		var id string
		if id, err = ensureID(name); !errors.Is(err, errIDRecoveryHeld) {
			return id, err
		}
	}
	return "", err
}

func ensureID(name string) (string, error) {
	switch id, err := readID(name); {
	case err == nil:
		return id, nil
	case os.IsNotExist(err):
		// Nobody has made one yet, or nobody had a moment ago - the create below is
		// what settles which.
	default:
		// There is a file and it is not an identity, which is most often one another
		// agent has just created and not yet written: O_EXCL publishes the name before
		// the contents. Waited out rather than refused, because refusing turns the
		// volume off for this agent over a window one write wide.
		return awaitID(name)
	}

	var buf [idBytes]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf[:])

	// Created with O_EXCL, not written elsewhere and linked into place.
	//
	// A hard link is the tidier create-if-absent - the name appears already holding its
	// contents - but link(2) is not implemented by every filesystem a ReadWriteMany
	// claim can be backed by, SMB and some CSI drivers among them. ReadWriteMany
	// describes who may mount a volume, not what the filesystem under it can do, so
	// requiring a link turned the cache off on claims that were otherwise perfectly
	// good, and silently: EnsureID failed, the volume was refused, and every cache fell
	// back to the object store.
	//
	// O_EXCL is as exclusive and asks for nothing unusual. What it costs is that the
	// name exists before its contents do, which awaitID below covers.
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, SharedFileMode)
	if err != nil {
		if os.IsExist(err) {
			// Another agent created it first. Two identities for one volume would
			// split its cache in half, so the loser takes the winner's.
			return awaitID(name)
		}
		return "", err
	}
	if err := writeID(f, id); err != nil {
		return "", err
	}
	return id, nil
}

// writeID sets the mode and then the contents, in that order, and closes the file.
//
// OpenFile's mode is filtered by the umask, so it has to be set explicitly: an agent
// under 0077 would otherwise publish the identity 0600 and own it. Every other agent on
// the volume reads this - that is the whole point of it - and one that cannot gets no
// identity, which turns the volume off for it entirely, so a single strict umask would
// take the feature away from everyone else.
//
// Set before the contents rather than after, because an agent that dies in between
// leaves whatever it has already written. After, that is a *non-empty* 0600 identity,
// which no other agent can read and which the repair below deliberately will not touch -
// a file with contents is somebody's identity, not a half-made one - so the volume is
// off for everyone else permanently. Before, the worst a death leaves is the empty file
// the repair exists for.
func writeID(f *os.File, id string) error {
	// On the file rather than the name: the name could by then be a different file.
	if err := f.Chmod(SharedFileMode); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.WriteString(id + "\n"); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// awaitID reads an identity another agent is in the middle of creating.
//
// O_EXCL publishes the name before the contents, so a reader arriving in between finds
// it empty. That window is one write wide and happens once in a volume's life, but
// falling at it would refuse the volume for this agent - so it is waited out rather
// than reported.
//
// A write that never comes at all leaves the same empty file permanently. The agent that
// created it was killed in the window - an eviction or a node failure between two
// syscalls - and nothing afterwards repairs it: every later startup waits out the
// attempts, finds it still empty, and turns the volume off for the whole installation,
// over six bytes. So an empty identity old enough not to be anybody's open write is
// removed and the create attempted again.
func awaitID(name string) (string, error) {
	var err error
	for attempt := 0; attempt < idReadAttempts; attempt++ {
		var id string
		if id, err = readID(name); err == nil {
			return id, nil
		}
		if os.IsNotExist(err) {
			// Removed again between the failed create and here, which is not a race
			// this has to win: the next pass through EnsureID makes it.
			return "", err
		}
		time.Sleep(idReadDelay)
	}
	id, recoverErr := recoverStaleID(name)
	switch {
	case recoverErr == nil:
		return id, nil
	case errors.Is(recoverErr, errIDRecoveryHeld):
		// Passed through rather than folded into the error below, because it is the one
		// the caller acts on: another agent is writing the identity right now, so the
		// answer is to read again.
		return "", recoverErr
	}
	// Anything else leaves the original failure, which says what was actually wrong
	// with the identity rather than what the repair made of it.
	return "", err
}

// recoverStaleID fills in an identity that was created and never written.
//
// **The file is never unlinked, on any path.** Removing it and creating a fresh one is
// the obvious repair and is not safe: the decision to remove rests on a stat taken
// earlier, so two agents repairing together can have one remove the file, create its
// own and write it, and the other then unlink *that* - leaving each with an identity
// the other has never seen, and the volume's cache split between two key prefixes until
// both restart. Nothing in POSIX unlinks a particular file rather than a name, so the
// repair writes into the file that is already there, which belongs to whoever reaches
// it. A late arrival finds contents and simply reads them.
//
// The marker is what keeps two agents from writing different identities into it. Both
// agents reach here having waited out the same attempts, so they are not phased apart
// by chance: a race here is likely rather than remote. Only the agent that creates the
// marker with O_EXCL repairs anything, and it holds it until the identity is written.
func recoverStaleID(name string) (string, error) {
	marker, err := acquireIDRecovery(filepath.Join(filepath.Dir(name), idRecoveryName))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = marker.Close()
		// Removed by the agent that created it, which is the only one that may.
		_ = os.Remove(marker.Name())
	}()

	// Looked at again under the marker, because the wait for it may have been spent on
	// another agent doing exactly this repair - whose identity is then the one to take.
	info, err := os.Stat(name)
	if err != nil {
		return "", err
	}
	if info.Size() != 0 {
		return readID(name)
	}
	if time.Since(info.ModTime()) < idStaleAfter {
		// Young enough that the write may still be on its way, so it is left alone and
		// the next startup looks again. Clock skew only makes it look younger than it
		// is, which errs towards leaving it.
		return "", fmt.Errorf("%s is empty", name)
	}

	var buf [idBytes]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf[:])

	// Opened without O_TRUNC and without O_EXCL: the file is the one already there, and
	// it is empty.
	f, err := os.OpenFile(name, os.O_WRONLY, SharedFileMode)
	if err != nil {
		return "", err
	}
	if err := writeID(f, id); err != nil {
		return "", err
	}
	return id, nil
}

// acquireIDRecovery takes the right to repair the identity, or reports that another
// agent holds it.
func acquireIDRecovery(marker string) (*os.File, error) {
	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, SharedFileMode)
	if err == nil || !os.IsExist(err) {
		return f, err
	}

	// Either another agent is repairing right now, or one died in the middle of it. The
	// second would block every repair from here on, which is the failure this whole path
	// exists to undo, one level up - so a marker old enough not to be anybody's is
	// cleared and the create tried once more. A held marker is seconds old, so clearing
	// one that is genuinely held takes a crash to have happened first, and costs at
	// worst the split identity the marker is here to prevent.
	info, statErr := os.Stat(marker)
	if statErr != nil || time.Since(info.ModTime()) < idStaleAfter {
		return nil, errIDRecoveryHeld
	}
	if rmErr := os.Remove(marker); rmErr != nil && !os.IsNotExist(rmErr) {
		return nil, errIDRecoveryHeld
	}
	return os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, SharedFileMode)
}

func readID(name string) (string, error) {
	body, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(body))
	if id == "" {
		return "", fmt.Errorf("%s is empty", name)
	}
	return id, nil
}

// ScopedKey partitions a cache key by the volume it will be stored on.
//
// Applied to the restore keys as well, and as a prefix rather than a suffix, so that
// prefix matching stays inside one volume: a restore key of "npm-" must not match
// another volume's "npm-abc", which would be an entry this runner cannot read.
//
// An empty key is left empty. The Control Plane ignores an empty restore key on
// purpose; scoped it would become "<id>/", which is a prefix every entry on this
// volume matches - so a workflow carrying one, from a template that resolved to
// nothing, would restore whichever entry happened to be newest.
func ScopedKey(id, key string) string {
	if id == "" || key == "" {
		return key
	}
	return id + "/" + key
}
