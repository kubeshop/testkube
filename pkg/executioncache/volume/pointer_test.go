package volume

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeRoundTrips(t *testing.T) {
	in := Pointer{Path: "inbox/abc123/deadbeef", Size: 1 << 30}

	out, ok := Decode(Encode(in))

	require.True(t, ok)
	assert.Equal(t, in, out)
}

// An archive and a pointer arrive the same way - as the body of a cache object - so the
// only thing telling them apart is the first few bytes. A gzip stream read as a pointer
// would be an unpack that never happens.
func TestDecodeLeavesAnArchiveAlone(t *testing.T) {
	var archive bytes.Buffer
	w := gzip.NewWriter(&archive)
	_, err := w.Write([]byte("not a pointer"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	_, ok := Decode(archive.Bytes())

	assert.False(t, ok, "a gzip body must not be read as a pointer")
}

func TestDecodeRefusesAnUnreadablePointer(t *testing.T) {
	for name, body := range map[string][]byte{
		"magic alone":     []byte(Magic),
		"truncated json":  []byte(Magic + `{"p":"inbox/a/b"`),
		"not json at all": []byte(Magic + "gzip bytes would go here"),
		"no path":         []byte(Magic + `{"s":10}`),
		"short of magic":  []byte("tkcache"),
		"empty":           nil,
	} {
		t.Run(name, func(t *testing.T) {
			_, ok := Decode(body)
			assert.False(t, ok)
		})
	}
}

// A pointer is written by a pod, so the path in it is input. These are the shapes that
// would let one name something other than an entry this store wrote.
func TestValidatePathRefusesAnythingButAnEntryInAnInbox(t *testing.T) {
	for name, p := range map[string]string{
		"empty":               "",
		"traversal":           "inbox/abc/../../etc/passwd",
		"leading traversal":   "../inbox/abc/entry",
		"absolute":            "/inbox/abc/entry",
		"outside the inbox":   "elsewhere/abc/entry",
		"no resource segment": "inbox/entry",
		"too deep":            "inbox/abc/def/entry",
		"the inbox itself":    "inbox",
		"a whole resource":    "inbox/abc",
		"unclean":             "inbox/./abc/entry",
		"trailing slash":      "inbox/abc/entry/",
	} {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, ValidatePath(p), "%q must be refused", p)
		})
	}
}

func TestValidatePathAcceptsWhatTheStoreWrites(t *testing.T) {
	assert.NoError(t, ValidatePath("inbox/abc123/deadbeef"))
}

// The cap is what stops a body that merely starts with the magic from making a reader
// buffer an archive looking for the end of the JSON.
func TestDecodeDoesNotReadPastTheCap(t *testing.T) {
	body := []byte(Magic + `{"p":"inbox/a/b","s":1,"pad":"` + strings.Repeat("x", MaxPointerBytes*2) + `"}`)

	_, ok := Decode(body)

	assert.False(t, ok, "a pointer larger than the cap is not one")
}
