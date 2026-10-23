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

	// idGenerations bounds how many identity files are tried before the volume is
	// refused.
	//
	// A new one is started only by an agent dying between creating the previous name
	// and writing thirteen bytes into it, a window two syscalls wide, and the volume
	// then carries the abandoned name for good. Eight is far past what that can
	// plausibly happen in.
	idGenerations = 8

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

// errIDAbandoned reports an identity that was created and never written, and has sat
// there too long for the write to still be coming. It never leaves this file.
var errIDAbandoned = errors.New("volume identity was created and never written")

// errIDSuperseded reports that a later generation appeared while this one was being
// written, so the identity just written is not the one the volume answers with. It never
// leaves this file.
var errIDSuperseded = errors.New("volume identity was superseded while being written")

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
//
// An identity that was created and never written - an agent killed between the two
// syscalls - is superseded rather than repaired, by moving to the next generation of the
// name. **Nothing here ever unlinks anything**, which is what makes concurrent agents
// safe: every name is settled by an O_EXCL create, which is atomic, and a name that has
// been used is never reused, so there is no instant at which one agent can remove what
// another has just created. Repairing in place needs exclusion, and a lock on a shared
// filesystem cannot be reclaimed safely - reclaiming it means unlinking a path on the
// strength of a stat taken earlier, which is the very race the lock was there to
// prevent, so a crash would either wedge the volume for good or let two agents write
// different identities into it.
func EnsureID(mountPath string) (string, error) {
	if mountPath == "" {
		return "", nil
	}

	var err error
	for generation := 0; generation < idGenerations; generation++ {
		// A later generation supersedes this one. It exists only because this one was
		// found abandoned, and the agents that made it are using it - so an abandoned
		// name that is somehow written long afterwards cannot take the volume back off
		// them.
		if _, statErr := os.Stat(idName(mountPath, generation+1)); statErr == nil {
			continue
		}

		var id string
		id, err = ensureID(mountPath, generation)
		if errors.Is(err, errIDAbandoned) || errors.Is(err, errIDSuperseded) {
			continue
		}
		return id, err
	}
	return "", err
}

// idName is where a generation of the identity lives. The first keeps the plain name,
// so a volume that never had one abandoned looks exactly as it always did.
func idName(mountPath string, generation int) string {
	if generation == 0 {
		return filepath.Join(mountPath, IDName)
	}
	return filepath.Join(mountPath, fmt.Sprintf("%s.%d", IDName, generation+1))
}

// ensureID settles one generation of the identity, and refuses to answer with it if that
// generation has been superseded.
//
// The check covers every way an identity is arrived at, not only writing one. An agent
// waiting out an empty generation holds no opinion for as long as it waits, and that wait
// is a second long: another agent can declare the generation abandoned and create its
// successor inside it, and the stalled creator can then fill the old name, so the waiter
// is handed an identity that was superseded while it waited. Reading an existing
// generation has a narrower form of the same gap, between the caller's look for a
// successor and the read.
func ensureID(mountPath string, generation int) (string, error) {
	id, err := settleID(mountPath, generation)
	if err != nil {
		return "", err
	}

	// A successor existing now is the one the volume answers with, whatever this
	// generation says. Paired with inheritedID, which covers a successor created after
	// this stat: it inherits whatever this generation holds by then, so the two cannot
	// disagree whichever order they fall in.
	if _, statErr := os.Stat(idName(mountPath, generation+1)); statErr == nil {
		return "", errIDSuperseded
	}
	return id, nil
}

func settleID(mountPath string, generation int) (string, error) {
	name := idName(mountPath, generation)

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

	// What goes in it is decided now the create is won, not before. The generation this
	// one supersedes was empty when that was decided, and may have been written since -
	// by the very agent whose stall made it look abandoned. Inheriting what it gained
	// is what keeps that agent's identity and this one the same.
	id, err := inheritedID(mountPath, generation)
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if err := writeID(f, id); err != nil {
		return "", err
	}

	// Whether this agent was itself the one declared abandoned is settled by the caller,
	// which checks for a successor on every path out rather than only this one.
	return id, nil
}

// inheritedID is what a new generation carries: the identity of the newest generation
// before it that has one, or a fresh identity when none has.
//
// Inherited rather than freshly generated so that a predecessor written late does not
// end up naming a different prefix from the successor that replaced it. An abandoned
// generation holds nothing, which is the ordinary case and the one that generates.
func inheritedID(mountPath string, generation int) (string, error) {
	for previous := generation - 1; previous >= 0; previous-- {
		if id, err := readID(idName(mountPath, previous)); err == nil {
			return id, nil
		}
	}

	var buf [idBytes]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
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
// over thirteen bytes. So an empty identity old enough not to be anybody's open write is
// reported as abandoned, and the caller moves to the next generation of the name.
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
	if abandonedID(name) {
		return "", errIDAbandoned
	}
	return "", err
}

// abandonedID reports an identity that was created and never written, and that nobody is
// going to write now.
//
// Only an empty file qualifies. One holding something readID could not make sense of is
// a volume being shared with a writer this agent does not understand, and superseding it
// would be the wrong answer: the cache turns off here rather than partitioning the
// volume against them.
func abandonedID(name string) bool {
	info, err := os.Stat(name)
	if err != nil || info.Size() != 0 {
		return false
	}
	// The mtime may have been set by the node that created the file rather than by the
	// server holding it, so clock skew can make a file look older than it is as easily
	// as younger - a skewed enough creator has its identity declared abandoned the
	// moment it is made. The window that opens is closed on both sides by ensureID
	// rather than by trusting this, which is why the margin here can stay small enough
	// to be useful.
	return time.Since(info.ModTime()) >= idStaleAfter
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
