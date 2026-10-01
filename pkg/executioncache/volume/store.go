package volume

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"
)

// EntryRoot is the directory inside an entry that mirrors the filesystem.
//
// A cached path is stored under it by its absolute path with the leading separator
// dropped, so /data/repo/node_modules lives at <entry>/root/data/repo/node_modules.
// That mirrors what the archive form did - a tarball rooted at "/" - which is what lets
// one entry hold several declared paths on different volumes, and keeps the restore's
// allowlist the same shape it already was.
const EntryRoot = "root"

// Store reads entries off the shared cache volume.
//
// It is the whole volume, mounted read-only. Reads are not confined to this execution
// because a restore legitimately reads what an earlier, unrelated execution saved -
// which is also why read confinement is the cost of sharing one volume rather than
// something this type could recover.
type Store struct {
	// root is the mount, opened once. Every path below is resolved inside it, so a
	// pointer cannot name a directory outside the volume however it is spelled.
	root *os.Root
}

// Inbox writes entries into the one directory this execution may write to.
//
// The directory is the volume's inbox/<resource id>, reached through a subPath mount,
// so the confinement is kubelet's rather than this type's: what the pod holds is that
// directory and nothing else. Nothing here re-checks it, because a check in this
// process would only bind the code that calls it - the step's own command holds the
// same mount.
type Inbox struct {
	// dir is the subPath mount, which already *is* inbox/<resource id>.
	dir *os.Root
	// path is the mount path, for the staging directory's absolute name.
	path string
	// rel is what that directory is called from the volume root, so that a pointer can
	// name the entry for a reader that mounted the whole volume instead.
	rel string
}

// OpenStore returns the read side of the volume, or nil and the reason there is none.
//
// Nil is the whole fallback mechanism. Every caller treats it as "use the object
// store", so a volume that is absent or unreadable degrades to exactly the behaviour
// that shipped before the volume existed. The reason is returned rather than logged so
// the caller can print it in its own voice; a silent absence would leave an operator
// unable to tell a cold cache from a misconfigured mount.
func OpenStore(mountPath string) (*Store, string) {
	if mountPath == "" {
		return nil, "no shared cache volume is configured"
	}
	root, err := os.OpenRoot(mountPath)
	if err != nil {
		return nil, fmt.Sprintf("cannot read the shared cache volume at %s (%v)", mountPath, err)
	}
	return &Store{root: root}, ""
}

// OpenInbox returns the write side, or nil and the reason there is none.
//
// It probes with a real create-and-remove rather than a stat or a permission check,
// because a mount that cannot be written to - read-only, full, squashed to an id with
// no access - announces itself only at the first write, which would otherwise be after
// copying a tree that can run to gigabytes.
func OpenInbox(mountPath, resourceID string) (*Inbox, string) {
	if mountPath == "" {
		return nil, "no shared cache volume is configured"
	}
	if resourceID == "" {
		// Without one, a committed entry cannot be named from the volume root, so no
		// reader could ever find it. This cannot happen for a real execution; it guards
		// against the config being threaded through without the id.
		return nil, "this step has no resource id, so its cache entry could not be named"
	}

	dir, err := os.OpenRoot(mountPath)
	if err != nil {
		return nil, fmt.Sprintf("cannot open the shared cache inbox at %s (%v)", mountPath, err)
	}

	in := &Inbox{dir: dir, path: mountPath, rel: path.Join(InboxDir, resourceID)}
	if err := in.probe(); err != nil {
		dir.Close()
		return nil, fmt.Sprintf("cannot write to the shared cache inbox at %s (%v)", mountPath, err)
	}
	return in, ""
}

func (in *Inbox) probe() error {
	name, err := randomName()
	if err != nil {
		return err
	}
	f, err := in.dir.Create(name + ".probe")
	if err != nil {
		return err
	}
	f.Close()
	return in.dir.Remove(name + ".probe")
}

// Close releases the mount handle.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	return s.root.Close()
}

// Close releases the mount handle.
func (in *Inbox) Close() error {
	if in == nil {
		return nil
	}
	return in.dir.Close()
}

// OpenEntry opens the mirrored tree inside an entry, for reading.
//
// The returned root is <entry>/root, so every path under it is an absolute path with
// its leading separator dropped - the same shape the caller restores to.
//
// The pointer came out of an object a pod wrote, so it is validated here rather than
// where it was decoded: this is the only place that turns one into a directory, and a
// validation that lives anywhere else can be bypassed by a second caller.
func (s *Store) OpenEntry(p Pointer) (*os.Root, error) {
	if err := ValidatePath(p.Path); err != nil {
		return nil, err
	}
	return s.root.OpenRoot(path.Join(p.Path, EntryRoot))
}

// Stage creates the directory an entry is built in, and returns its absolute path
// along with the name Commit needs.
//
// It is named .tmp so that the sweep, and a later step of this same execution, can tell
// a copy that is still running from one that finished. Commit renames it; nothing else
// ever publishes a name a reader could follow.
func (in *Inbox) Stage() (dir string, name string, err error) {
	base, err := randomName()
	if err != nil {
		return "", "", err
	}
	name = base + ".tmp"
	if err := in.dir.Mkdir(name, 0o777); err != nil {
		return "", "", err
	}
	if err := in.dir.Mkdir(path.Join(name, EntryRoot), 0o777); err != nil {
		return "", "", err
	}
	return path.Join(in.path, name, EntryRoot), name, nil
}

// Commit makes a staged entry readable and returns the pointer naming it.
//
// The ordering is the atomicity argument, and it is shorter than it would be if
// immutability lived here:
//
//	(caller) copy - the tree is written under a .tmp name no reader follows
//	rename        - within one directory on one filesystem, so it is atomic; a reader
//	                either finds the whole entry under its published name or nothing
//	(caller) PUT  - the pointer is published last, conditionally, which is what
//	                resolves a race between two executions saving the same key
//
// So no reader can observe a half-copied tree: the name it would have to open does not
// exist until the copy has finished, and the pointer that would name it does not exist
// until the rename has happened.
func (in *Inbox) Commit(staged string, size int64) (Pointer, error) {
	name, err := randomName()
	if err != nil {
		return Pointer{}, err
	}
	if err := in.dir.Rename(staged, name); err != nil {
		return Pointer{}, err
	}
	return Pointer{Path: path.Join(in.rel, name), Size: size}, nil
}

// DiscardStaged removes a staging directory that was never committed.
func (in *Inbox) DiscardStaged(staged string) {
	if in == nil || staged == "" {
		return
	}
	_ = in.removeAll(staged)
}

// Discard removes an entry whose pointer was never published.
//
// Every path that does not publish has to call this: a lost race, an entry that already
// existed, a refused or failed upload. The entry is reachable only through its pointer,
// so one left behind is invisible until the volume fills, and it can be gigabytes.
// Failures are ignored because there is nothing useful to do about them and the sweep
// collects the remainder.
func (in *Inbox) Discard(p Pointer) {
	if in == nil || p.Path == "" {
		return
	}
	if err := ValidatePath(p.Path); err != nil {
		return
	}
	// A pointer this inbox did not write names an entry it cannot reach anyway, but
	// refusing it here keeps the mount's own confinement from being the only thing
	// standing between a malformed pointer and somebody else's entry.
	if !strings.HasPrefix(p.Path, in.rel+"/") {
		return
	}
	_ = in.removeAll(path.Base(p.Path))
}

// removeAll deletes a directory tree inside the inbox.
//
// os.Root has no RemoveAll, so this walks it: every step stays inside the root, which
// is what keeps a bad name from deleting outside the mount.
func (in *Inbox) removeAll(name string) error {
	f, err := in.dir.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		// Not a directory - remove it as a plain file.
		return in.dir.Remove(name)
	}
	for _, e := range entries {
		child := path.Join(name, e.Name())
		if e.IsDir() {
			if err := in.removeAll(child); err != nil {
				return err
			}
			continue
		}
		if err := in.dir.Remove(child); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return in.dir.Remove(name)
}

// randomName returns an unguessable name.
//
// Random rather than sequential because two cached steps of one execution share an
// inbox, and a collision would have one overwrite the other's entry.
func randomName() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
