package convert

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEscapeCopyValue(t *testing.T) {
	t.Parallel()

	// The NUL byte is built rather than written literally so this file stays
	// free of control characters.
	nul := string([]byte{0x00})

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"empty becomes null", "", `\N`},
		{"tab", "a\tb", `a\tb`},
		{"newline", "a\nb", `a\nb`},
		{"carriage return", "a\rb", `a\rb`},
		{"backslash", `a\b`, `a\\b`},
		{"nul is dropped", "a" + nul + "b", "ab"},
		{"backslash before N is not a null sentinel", `\N`, `\\N`},
		{"all specials at once", "a\tb\nc\rd\\e", `a\tb\nc\rd\\e`},
		{"unicode is preserved", "café ✓", "café ✓"},
		{"escaped dot is preserved", "app．kubernetes．io", "app．kubernetes．io"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, escapeCopyValue(tt.in))
		})
	}
}

// A value that escapes to the literal characters backslash-N must not be read
// back as SQL NULL, otherwise a legitimate string would silently become NULL.
func TestEscapeCopyValueDistinguishesNullSentinel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `\N`, escapeCopyValue(""), "empty string must serialize as the null sentinel")
	assert.NotEqual(t, `\N`, escapeCopyValue(`\N`), "the literal string must not collide with the sentinel")
}

func TestEscapeJSONB(t *testing.T) {
	t.Parallel()

	nul := string([]byte{0x00})

	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"nil becomes null", nil, `\N`},
		{"empty becomes null", []byte{}, `\N`},
		{"json null becomes sql null", []byte("null"), `\N`},
		{"object", []byte(`{"a":1}`), `{"a":1}`},
		{"empty object is kept", []byte(`{}`), `{}`},
		{"empty array is kept", []byte(`[]`), `[]`},
		{"embedded quotes are untouched", []byte(`{"a":"b\"c"}`), `{"a":"b\\"c"}`},
		{"literal nul is dropped", []byte(`{"a":"b` + nul + `c"}`), `{"a":"bc"}`},
		{"tab inside json is escaped", []byte("{\"a\":\"b\tc\"}"), `{"a":"b\tc"}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, escapeJSONB(tt.in))
		})
	}
}

// PostgreSQL rejects NUL in JSONB both as a raw byte (SQLSTATE 22021) and as the
// six-character unicode escape encoding/json produces (SQLSTATE 22P05).
func TestSanitizeJSONForPGStripsBothNullForms(t *testing.T) {
	t.Parallel()

	escaped := `{"a":"b` + string(jsonNullEscape) + `c"}`
	assert.Equal(t, `{"a":"bc"}`, string(sanitizeJSONForPG([]byte(escaped))),
		"the unicode escape form must be stripped")

	raw := append([]byte(`{"a":"b`), 0x00)
	raw = append(raw, []byte(`c"}`)...)
	assert.Equal(t, `{"a":"bc"}`, string(sanitizeJSONForPG(raw)),
		"the raw byte form must be stripped")
}

func TestJSONNullEscapeSpelling(t *testing.T) {
	t.Parallel()

	// Guards the byte-wise spelling in copyutil.go against a typo. The expected
	// value is what encoding/json emits for a NUL byte, derived here rather than
	// written as a literal so the two spellings cannot drift together.
	encoded, err := json.Marshal(string([]byte{0x00}))
	assert.NoError(t, err)
	assert.Equal(t, `"`+string(jsonNullEscape)+`"`, string(encoded))
}

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("zero time becomes null", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `\N`, formatTimestamp(time.Time{}))
	})

	t.Run("utc", func(t *testing.T) {
		t.Parallel()
		ts := time.Date(2026, 8, 26, 13, 45, 30, 123456000, time.UTC)
		assert.Equal(t, "2026-08-26 13:45:30.123456+00", formatTimestamp(ts))
	})

	t.Run("offset zone is carried", func(t *testing.T) {
		t.Parallel()
		ts := time.Date(2026, 8, 26, 13, 45, 30, 0, time.FixedZone("CEST", 2*60*60))
		assert.Equal(t, "2026-08-26 13:45:30+02", formatTimestamp(ts))
	})

	t.Run("sub-microsecond precision is truncated", func(t *testing.T) {
		t.Parallel()
		ts := time.Date(2026, 8, 26, 13, 45, 30, 999999999, time.UTC)
		// PostgreSQL stores microseconds; the nanosecond remainder is dropped
		// rather than rounding the second up.
		assert.Equal(t, "2026-08-26 13:45:30.999999+00", formatTimestamp(ts))
	})
}

func TestToJSONBytes(t *testing.T) {
	t.Parallel()

	t.Run("nil marshals to json null", func(t *testing.T) {
		t.Parallel()
		b, err := toJSONBytes(nil)
		assert.NoError(t, err)
		assert.Equal(t, "null", string(b))
	})

	t.Run("typed nil map marshals to json null", func(t *testing.T) {
		t.Parallel()
		var m map[string]string
		b, err := toJSONBytes(m)
		assert.NoError(t, err)
		assert.Equal(t, "null", string(b))
		assert.Equal(t, `\N`, escapeJSONB(b), "a nil map must round-trip to SQL NULL")
	})

	t.Run("empty map is distinct from nil", func(t *testing.T) {
		t.Parallel()
		b, err := toJSONBytes(map[string]string{})
		assert.NoError(t, err)
		assert.Equal(t, `{}`, escapeJSONB(b),
			"an empty map must stay an empty object, not become NULL")
	})
}
