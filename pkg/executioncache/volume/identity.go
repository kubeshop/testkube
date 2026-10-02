package volume

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	if id, err := readID(name); err == nil {
		return id, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	var buf [idBytes]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf[:])

	// Written through a temporary name and renamed, so a second agent starting at the
	// same moment cannot read a half-written identity. The loser of that race re-reads
	// rather than overwriting: two identities for one volume would split its cache in
	// half for no reason.
	tmp := name + ".tmp-" + id
	if err := os.WriteFile(tmp, []byte(id+"\n"), 0o666); err != nil {
		return "", err
	}
	defer os.Remove(tmp)
	if err := os.Link(tmp, name); err != nil {
		if existing, readErr := readID(name); readErr == nil {
			return existing, nil
		}
		return "", err
	}
	return id, nil
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
