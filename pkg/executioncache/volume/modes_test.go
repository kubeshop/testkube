package volume

import (
	"io/fs"
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
