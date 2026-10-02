package volume

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Only what differs from the defaults is recorded, which is what keeps the manifest
// small enough to be worth having: a dependency tree is almost entirely 0644 and 0755.
func TestModesRecordsOnlyTheExceptions(t *testing.T) {
	m := Modes{}

	assert.False(t, m.Record("pkg/index.js", 0o644, false))
	assert.False(t, m.Record("pkg", 0o755, true))
	assert.True(t, m.Record("pkg/.npmrc", 0o600, false))
	assert.True(t, m.Record("pkg/bin/tool", 0o755, false), "a file at the directory default is still an exception")
	assert.True(t, m.Record("private", 0o700, true))

	assert.Equal(t, Modes{
		"pkg/.npmrc":   0o600,
		"pkg/bin/tool": 0o755,
		"private":      0o700,
	}, m)
}

func TestModesRoundTrip(t *testing.T) {
	m := Modes{"a/.ssh/id_rsa": 0o600, "a/bin/run": 0o755, "ro": 0o444}

	got := DecodeModes(m.Encode())

	assert.Equal(t, m, got)
}

// Records end at a NUL because a newline is a character a filename may hold, and the
// paths here are whatever a workflow cached.
func TestModesRoundTripAPathHoldingANewline(t *testing.T) {
	m := Modes{"weird\nname": 0o600}

	assert.Equal(t, m, DecodeModes(m.Encode()))
}

// The manifest is an optimisation over restoring at the defaults, so one record it
// cannot read costs that path its mode rather than the whole restore.
func TestDecodeModesSkipsWhatItCannotRead(t *testing.T) {
	got := DecodeModes([]byte("0600 keep\x00not-a-mode nope\x00\x00garbage\x000755 also/keep\x00"))

	assert.Equal(t, Modes{"keep": 0o600, "also/keep": 0o755}, got)
}

func TestDecodeModesOfNothingIsEmpty(t *testing.T) {
	assert.Empty(t, DecodeModes(nil))
	assert.Empty(t, Modes{}.Encode())
}

// Deepest first, so that narrowing a directory cannot shut the restore out of what is
// still inside it.
func TestModesApplyIsDeepestFirst(t *testing.T) {
	m := Modes{"a": 0o700, "a/b/c": 0o600, "a/b": 0o700, "z": fs.FileMode(0o600)}

	assert.Equal(t, []string{"a/b/c", "a/b", "a", "z"}, m.Apply())
}

// The manifest is read before any copy limit applies, and the entry it comes from is
// not necessarily one this installation wrote: the inbox is mounted into the step's own
// container, so a workflow can commit an entry carrying a manifest of any size and
// every later execution restoring that key would read it.
func TestDecodeModesFromKeepsNoMoreThanTheEntryLimit(t *testing.T) {
	var manifest strings.Builder
	for i := 0; i < 5_000; i++ {
		fmt.Fprintf(&manifest, "0600 path/%d\x00", i)
	}

	got := DecodeModesFrom(strings.NewReader(manifest.String()), 10)

	assert.Len(t, got, 10, "an entry cannot hold more exceptional paths than it holds paths")
}

// A manifest with no terminator anywhere in it must not be read into memory entire
// while something looks for one.
func TestDecodeModesFromRefusesAnEndlessRecord(t *testing.T) {
	endless := "0600 " + strings.Repeat("a", maxModeRecordBytes*4)

	got := DecodeModesFrom(strings.NewReader(endless), 100)

	assert.Empty(t, got, "a record longer than the buffer costs that path its mode, nothing more")
}

// The records before an endless one are still read: the scanner stops where it cannot
// go on, and what it already had stands.
func TestDecodeModesFromKeepsWhatItReadBeforeAnEndlessRecord(t *testing.T) {
	data := "0600 first\x00" + "0700 " + strings.Repeat("b", maxModeRecordBytes*2)

	got := DecodeModesFrom(strings.NewReader(data), 100)

	assert.Equal(t, Modes{"first": 0o600}, got)
}

// A last record whose terminator is missing is still a record.
func TestDecodeModesFromReadsAnUnterminatedLastRecord(t *testing.T) {
	got := DecodeModesFrom(strings.NewReader("0600 a\x000700 b"), 100)

	assert.Equal(t, Modes{"a": 0o600, "b": 0o700}, got)
}

// countingReader reports how much of a manifest was actually read.
type countingReader struct {
	inner io.Reader
	read  int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.inner.Read(p)
	c.read += n
	return n, err
}

// Budgeting the paths kept let a repeated record cost nothing, so a manifest of one
// short record written over and over was read to its end however long that was. Memory
// stayed bounded; the scan and the reads off the volume did not - and the entry it came
// from is one a workflow wrote, restored by an execution that had no part in it.
func TestDecodeModesFromStopsReadingAfterTheRecordLimit(t *testing.T) {
	manifest := strings.Repeat("0600 a\x00", 100_000)
	counted := &countingReader{inner: strings.NewReader(manifest)}

	got := DecodeModesFrom(counted, 10)

	assert.Equal(t, Modes{"a": 0o600}, got, "every record names the same path")
	assert.Less(t, counted.read, len(manifest)/2,
		"the limit has to bound the reading, not only what is kept")
}

// Malformed records spend the budget too: they are as cheap to write and as expensive
// to scan.
func TestDecodeModesFromCountsRecordsItCannotRead(t *testing.T) {
	manifest := strings.Repeat("rubbish\x00", 100_000) + "0600 real\x00"
	counted := &countingReader{inner: strings.NewReader(manifest)}

	got := DecodeModesFrom(counted, 10)

	assert.Empty(t, got, "the budget was spent before the real record arrived")
	assert.Less(t, counted.read, len(manifest)/2)
}

// The record count bounds how many paths are kept, not how long they are. At the entry
// limit, with every path as long as one may be, a manifest written to the letter of
// both caps still retained gigabytes of strings - before a byte of the tree was copied,
// so the copy limits never got a say and one entry could exhaust another's restore.
func TestDecodeModesFromBoundsThePathsItKeeps(t *testing.T) {
	// Each record is close to the per-record cap, so the byte budget is what has to
	// stop this rather than the count.
	long := strings.Repeat("p", 4000)
	var manifest strings.Builder
	for i := 0; i < 20_000; i++ {
		fmt.Fprintf(&manifest, "0600 %s/%d\x00", long, i)
	}

	got := DecodeModesFrom(strings.NewReader(manifest.String()), 1_000_000)

	kept := 0
	for rel := range got {
		kept += len(rel)
	}
	assert.LessOrEqual(t, kept, maxModeTotalBytes+4096,
		"a manifest cannot spend more of the restore's memory than this")
	assert.Less(t, len(got), 20_000, "and the records past the budget keep the defaults")
}
