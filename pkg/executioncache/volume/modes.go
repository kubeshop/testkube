package volume

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	// ModesName is the file beside an entry's tree recording the permissions its own
	// files cannot carry.
	//
	// A file on the volume is written by one execution and read by another, which may
	// run as a different user, so what is stored has to be readable by anyone - see
	// copyOut. That loses the original: a key stored 0600 becomes 0666, and restoring
	// it that way breaks the tools that check, ssh refusing a key that anyone could
	// read being the usual one. The archive backend has no such problem, carrying the
	// mode in each tar header, so the same workflow behaved differently depending on
	// which backend held its cache.
	//
	// Versioned in the name rather than inside: an entry written by a newer agent is
	// read by an older one on the same volume, and a name it does not know is a file
	// it ignores, which degrades to exactly the behaviour it had before this existed.
	ModesName = "modes.v1"

	// DefaultFileMode and DefaultDirMode are what a restore creates with, and so what
	// does not need recording. Nearly everything in a dependency tree is one or the
	// other, which is what keeps the file small enough to be worth having.
	DefaultFileMode = fs.FileMode(0o644)
	DefaultDirMode  = fs.FileMode(0o755)
)

// Modes records the permissions of the paths in an entry that differ from the defaults,
// keyed by path relative to the entry's tree.
type Modes map[string]fs.FileMode

// Record notes a path whose mode a restore would otherwise get wrong, and says whether
// it had to.
func (m Modes) Record(rel string, mode fs.FileMode, isDir bool) bool {
	perm := mode.Perm()
	if isDir && perm == DefaultDirMode {
		return false
	}
	if !isDir && perm == DefaultFileMode {
		return false
	}
	m[rel] = perm
	return true
}

// Encode writes the manifest.
//
// Records are terminated by a NUL rather than a newline, because a newline is a
// character a filename may contain and a NUL is not. One malformed record would
// otherwise take the rest of the manifest with it, and the paths that reach here are
// whatever a workflow cached.
func (m Modes) Encode() []byte {
	if len(m) == 0 {
		return nil
	}
	var out bytes.Buffer
	if err := m.EncodeTo(&out); err != nil {
		return nil
	}
	return out.Bytes()
}

// EncodeTo writes the manifest to w.
func (m Modes) EncodeTo(w io.Writer) error {
	// Sorted so that an entry's manifest is the same bytes however the walk ordered
	// it, which makes one diffable against another when something has gone wrong.
	paths := make([]string, 0, len(m))
	for rel := range m {
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		if _, err := fmt.Fprintf(w, "%04o %s\x00", m[rel].Perm(), rel); err != nil {
			return err
		}
	}
	return nil
}

// Recorder collects a tree's exceptional permissions as it is walked, under the same
// cap the restore reads them back under.
//
// Bounded for the same reason the decode is, and against the same number. The walk
// feeding this is bounded only by the entry limit, and 500,000 paths of up to PATH_MAX
// each is gigabytes of strings held before a single copy limit applies - where the
// archive backend streams each mode into a tar header and retains none of them. This
// runs in the step's own container, so being killed for it fails the step, and a cache
// may never do that.
//
// The budget is deliberately the decode's: a record past it would be dropped by every
// restore that read it, DecodeModesFrom stopping at the same total, so keeping one
// spends memory on something no restore will ever use. Past it a path keeps the
// defaults, which is the degradation this file already has for a record it cannot read.
type Recorder struct {
	modes Modes
	bytes int
}

func NewRecorder() *Recorder {
	return &Recorder{modes: make(Modes)}
}

// Record notes a path whose mode a restore would otherwise get wrong, while there is
// budget left to carry it.
func (r *Recorder) Record(rel string, mode fs.FileMode, isDir bool) {
	if r.bytes+len(rel) > maxModeTotalBytes {
		return
	}
	if r.modes.Record(rel, mode, isDir) {
		r.bytes += len(rel)
	}
}

// Empty reports that every path took a default, so there is no manifest to write.
func (r *Recorder) Empty() bool {
	return len(r.modes) == 0
}

// WriteModes writes the manifest beside an entry's tree.
//
// Streamed rather than encoded into a buffer and handed to os.WriteFile, which held a
// second copy of everything recorded - up to the budget again, on top of the map.
func WriteModes(name string, r *Recorder) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, SharedFileMode)
	if err != nil {
		return err
	}
	buffered := bufio.NewWriter(f)
	if err := r.modes.EncodeTo(buffered); err != nil {
		_ = f.Close()
		return err
	}
	if err := buffered.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// OpenFile's mode is filtered by the umask, and this is read by whichever agent
	// restores the entry - not necessarily the one that wrote it, nor as the same user.
	return os.Chmod(name, SharedFileMode)
}

// DecodeModes reads a manifest, ignoring any record it cannot make sense of.
//
// Tolerant on purpose. The manifest is an optimisation over restoring everything at the
// default, so a record that is malformed, or written by a version that says more than
// this one understands, costs one path its exact mode rather than the whole restore.
func DecodeModes(data []byte) Modes {
	return DecodeModesFrom(bytes.NewReader(data), 0)
}

// DefaultMaxModeRecords caps a manifest read with no entry limit of its own.
const DefaultMaxModeRecords = 1 << 20

// maxModeTotalBytes caps the paths a manifest leaves in memory.
//
// The record count does not: at the entry limit, with every path as long as one may be,
// a manifest written to the letter of both caps still retains gigabytes of strings. And
// it retains them before a byte of the tree is copied, so the copy limits never get a
// say - one entry could exhaust the restore of another.
//
// Sized for the legitimate case rather than the limit: the manifest holds only what
// differs from 0644 and 0755, which in a dependency tree is a handful of paths, and 32
// MiB is hundreds of thousands of realistic ones. A tree genuinely past it loses the
// modes of whatever follows, which is the degradation this file already has for a
// record it cannot read.
const maxModeTotalBytes = 32 << 20

// maxModeRecordBytes bounds one record, which is a mode, a space and a path. Generous
// against PATH_MAX, and what stops a manifest with no terminator anywhere in it from
// being read into memory entire while something looks for one.
const maxModeRecordBytes = 4096 + 64

// DecodeModesFrom reads a manifest without holding it in memory, keeping at most
// maxRecords of them. The limit is on records read, not on paths kept, so a repeated
// or malformed record spends it like any other. Zero takes DefaultMaxModeRecords, because an unbounded copy is
// the caller's choice about a tree it owns, where an unbounded manifest is an
// allocation sized by whoever wrote the entry.
//
// Bounded because the restore reads this before any of the copy limits apply, and the
// file is not necessarily one this installation wrote: the inbox an entry is built in
// is mounted into the step's own container, so a workflow can commit an entry carrying
// a manifest of any size it likes, and every later execution restoring that key would
// read it. Unbounded that is an out-of-memory in somebody else's restore, reached
// through a cache they had no part in writing.
//
// maxRecords is the entry limit for the same reason it bounds the tree: an entry cannot
// hold more exceptional paths than it holds paths. Records past it are dropped rather
// than failing the restore, which costs those paths their modes - the same degradation
// a manifest that cannot be parsed already gets.
func DecodeModesFrom(r io.Reader, maxRecords int) Modes {
	if maxRecords <= 0 {
		maxRecords = DefaultMaxModeRecords
	}

	modes := make(Modes)

	records := bufio.NewScanner(r)
	records.Buffer(make([]byte, 0, 4096), maxModeRecordBytes)
	records.Split(splitNUL)

	scanned, retained := 0, 0
	for records.Scan() {
		// Counted whatever the record turns out to be, rather than counting the paths
		// kept. Budgeting distinct valid paths left a repeated or malformed record
		// costing nothing, so a manifest of one short record written over and over was
		// read to its end however long that was - memory bounded, but the scan and the
		// reads off the volume were not, and the entry it came from is one a workflow
		// wrote.
		scanned++
		if scanned > maxRecords {
			break
		}

		record := records.Text()
		if record == "" {
			continue
		}
		mode, rel, ok := strings.Cut(record, " ")
		if !ok || rel == "" {
			continue
		}
		perm, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			continue
		}

		// Bounded by what is kept as well as by how much is read. The record count on
		// its own leaves the paths: at the entry limit, with every path as long as one
		// may be, a manifest crafted to the letter of both caps still retains gigabytes
		// of strings - and it retains them before a single byte of the tree is copied,
		// so the copy limits never get a say. Whatever is past the budget keeps the
		// default mode, which is the degradation this file already has for a record it
		// cannot read.
		retained += len(rel)
		if retained > maxModeTotalBytes {
			break
		}
		modes[rel] = fs.FileMode(perm).Perm()
	}
	// Scanner errors are not reported, including a record longer than the buffer: this
	// is an optimisation over restoring at the defaults, so what cannot be read costs
	// those paths their modes and nothing more.
	return modes
}

// splitNUL yields the NUL-terminated records Encode writes.
func splitNUL(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		// A manifest whose last record lost its terminator still has a record in it.
		return len(data), data, nil
	}
	return 0, nil, nil
}

// Apply returns the recorded paths in the order a restore has to set them: deepest
// first, so that narrowing a directory cannot stop the walk reaching what is inside it.
func (m Modes) Apply() []string {
	paths := make([]string, 0, len(m))
	for rel := range m {
		paths = append(paths, rel)
	}
	sort.Slice(paths, func(i, j int) bool {
		di, dj := strings.Count(paths[i], "/"), strings.Count(paths[j], "/")
		if di != dj {
			return di > dj
		}
		return paths[i] < paths[j]
	})
	return paths
}
