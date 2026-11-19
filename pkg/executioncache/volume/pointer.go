package volume

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

// errInvalidPath keeps every rejection in ValidatePath phrased the same way, so a log
// line names the pointer and what was wrong with it rather than only that it failed.
func errInvalidPath(why string) error {
	return fmt.Errorf("cache pointer path %s", why)
}

const (
	// Magic opens a pointer object.
	//
	// A body that does not begin with it is the archive itself, which is how an entry
	// written before the volume existed - or by a cluster that has none - still
	// restores. That makes the format its own fallback: a reader needs no configuration
	// to tell the two apart, only the first few bytes.
	Magic = "tkcache-pointer/v1\n"

	// MaxPointerBytes bounds what a reader buffers before deciding what it is holding.
	//
	// A pointer is a few hundred bytes. The cap exists so that a body which happens to
	// begin with the magic but is not a pointer cannot make a reader buffer an archive
	// into memory looking for the end of the JSON.
	MaxPointerBytes = 4 << 10

	// InboxDir is the only directory on the volume a pod ever writes into.
	InboxDir = "inbox"

	// EnvStorePath and EnvInboxPath tell the toolkit where the processor mounted the
	// shared volume, and are unset when it mounted none.
	//
	// Declared here, with the format, so that the half which mounts and the half which
	// looks cannot disagree: a processor that mounted a volume the toolkit did not look
	// for - or the reverse - would produce a cache that silently did nothing.
	//
	// They are environment rather than part of the encoded arguments because where the
	// cache is stored is a deployment fact. A workflow that could name it could name
	// somebody else's.
	EnvStorePath = "TK_CACHE_VOLUME"
	EnvInboxPath = "TK_CACHE_INBOX"

	// EnvInboxName is what the inbox mount is called from the volume root.
	//
	// The write mount is a subPath, so the pod sees its inbox as the mount itself and
	// cannot tell what it is called from the outside - but a pointer has to name the
	// entry for a reader that mounted the whole volume. The processor knows the
	// resource id when it builds the pod, so it resolves this there rather than leaving
	// the pod to work it out from its own environment.
	EnvInboxName = "TK_CACHE_INBOX_NAME"
)

// InboxFor is where one execution's entries live, relative to the volume root.
func InboxFor(resourceID string) string {
	return InboxDir + "/" + resourceID
}

// Pointer names the entry on the volume that an object stands in for.
type Pointer struct {
	// Path is relative to the volume root and is always "inbox/<resource>/<name>".
	//
	// It names a directory, not a file: an entry is the cached tree as it will be
	// restored, mirroring the filesystem from "/" the way the archive form did, so that
	// restoring is a copy rather than an unpack. Nothing is gzipped, which trades the
	// compression for per-file round trips - a trade worth measuring on a tree of many
	// small files, where the round trips can cost more than the compression saved.
	//
	// It is written by a pod, so it is validated on every read rather than trusted.
	// ValidatePath is what keeps it from naming anything but an entry, and resolving it
	// through os.Root is what keeps it from leaving the mount.
	Path string `json:"p"`

	// Size is the entry's total size in bytes.
	//
	// Carried because the object is now a few hundred bytes, so neither the restore's
	// reporting nor a future quota can read the size off the object any more.
	Size int64 `json:"s"`
}

// Encode renders a pointer as the body of a cache object.
func Encode(p Pointer) []byte {
	// Marshalling a struct of a string and an int64 cannot fail.
	body, _ := json.Marshal(p)
	out := make([]byte, 0, len(Magic)+len(body))
	out = append(out, Magic...)
	return append(out, body...)
}

// Decode reads a pointer from the head of a cache object's body.
//
// ok is false for a body that is not a pointer. That is not an error: it is an archive
// stored the old way, and the caller streams it as it always did. head may be shorter
// than the whole body - only the magic and the JSON after it are needed - but a caller
// that passes less than len(Magic) bytes of a pointer gets ok false, so read at least
// MaxPointerBytes before deciding.
func Decode(head []byte) (Pointer, bool) {
	if !bytes.HasPrefix(head, []byte(Magic)) {
		return Pointer{}, false
	}
	body := head[len(Magic):]
	if len(body) > MaxPointerBytes {
		body = body[:MaxPointerBytes]
	}

	var p Pointer
	if err := json.Unmarshal(body, &p); err != nil {
		// The magic matched but the rest did not parse. Treating this as "not a
		// pointer" would hand an unpacker a body beginning with the magic, which cannot
		// be a valid gzip stream either, so the caller gets a clearer failure by
		// learning the pointer was unreadable.
		return Pointer{}, false
	}
	if p.Path == "" {
		return Pointer{}, false
	}
	return p, true
}

// ValidatePath reports whether a pointer names an entry this store could have written.
//
// The shape is fixed - inbox/<resource>/<name> - rather than merely confined, because
// confinement alone would still let a pointer name any directory the mount carries. A
// pointer naming another execution's entry is a read, and reads across the volume are
// already the accepted cost of sharing one; pinning the shape is what keeps the surface
// to exactly that.
func ValidatePath(p string) error {
	if p == "" {
		return errInvalidPath("is empty")
	}
	if p != path.Clean(p) {
		return errInvalidPath("is not a clean path")
	}
	if path.IsAbs(p) {
		return errInvalidPath("is absolute")
	}

	parts := strings.Split(p, "/")
	if len(parts) != 3 {
		return errInvalidPath("is not inbox/<resource>/<name>")
	}
	if parts[0] != InboxDir {
		return errInvalidPath("does not start with " + InboxDir + "/")
	}
	for _, part := range parts[1:] {
		// path.Clean has already removed any "." and interior "..", so what remains to
		// refuse is a leading one and an empty segment.
		if part == "" || part == ".." || part == "." {
			return errInvalidPath("has an empty or relative segment")
		}
	}
	return nil
}
