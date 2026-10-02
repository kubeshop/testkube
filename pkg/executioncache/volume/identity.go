package volume

import (
	"crypto/rand"
	"encoding/hex"
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
)

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

	_, writeErr := f.WriteString(id + "\n")
	closeErr := f.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}

	// OpenFile's mode is filtered by the umask, so it is set explicitly: an agent under
	// 0077 would otherwise publish the identity 0600 and own it. Every other agent on
	// the volume reads this - that is the whole point of it - and one that cannot gets
	// no identity, which turns the volume off for it entirely. A single strict umask
	// would take the feature away from everyone else.
	if err := os.Chmod(name, SharedFileMode); err != nil {
		return "", err
	}
	return id, nil
}

// awaitID reads an identity another agent is in the middle of creating.
//
// O_EXCL publishes the name before the contents, so a reader arriving in between finds
// it empty. That window is one write wide and happens once in a volume's life, but
// falling at it would refuse the volume for this agent - so it is waited out rather
// than reported.
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
	return "", err
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
