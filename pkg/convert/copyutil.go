package convert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// copyNull is the sentinel the COPY statements below are configured to read as
// SQL NULL. It has to be spelled with a literal backslash, which is why every
// serializer writes `\N` rather than an empty field.
const copyNull = `\N`

// timestampLayout is the format PostgreSQL parses for TIMESTAMPTZ input in
// COPY text mode. The trailing -07 carries the zone offset.
const timestampLayout = "2006-01-02 15:04:05.999999-07"

// jsonNullEscape is how a NUL byte appears once encoding/json has escaped it:
// a backslash, a 'u', then four zeros. Spelled byte-wise so no literal NUL or
// escape sequence has to survive in this source file.
var jsonNullEscape = []byte{'\\', 'u', '0', '0', '0', '0'}

// toJSONBytes marshals v for a JSONB column. A nil value marshals to the JSON
// literal null, which escapeJSONB then turns into a SQL NULL.
func toJSONBytes(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v)
}

// escapeCopyValue renders a Go string as one COPY text field.
//
// Note that an empty string becomes SQL NULL, not an empty string. Callers
// writing a NOT NULL column must reject empty values before calling this --
// see validateExecution.
func escapeCopyValue(s string) string {
	if s == "" {
		return copyNull
	}
	return escapeCopySpecials(s)
}

// escapeJSONB renders marshaled JSON as one COPY text field.
func escapeJSONB(data []byte) string {
	if len(data) == 0 || string(data) == "null" {
		return copyNull
	}
	return escapeCopySpecials(string(sanitizeJSONForPG(data)))
}

// escapeCopySpecials escapes the four characters that terminate or misalign a
// COPY text field, and drops NUL outright.
func escapeCopySpecials(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}

// sanitizeJSONForPG removes null bytes, which PostgreSQL rejects both in TEXT
// (SQLSTATE 22021) and, as a unicode escape, in JSONB (SQLSTATE 22P05).
func sanitizeJSONForPG(data []byte) []byte {
	data = bytes.ReplaceAll(data, jsonNullEscape, nil)
	return bytes.ReplaceAll(data, []byte{0x00}, nil)
}

// formatTimestamp renders t for a TIMESTAMPTZ column, mapping the zero time to
// SQL NULL rather than to year 1.
func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return copyNull
	}
	return t.Format(timestampLayout)
}

// copyFrom streams r into tableName via COPY FROM STDIN. An empty reader is
// valid and copies zero rows.
func copyFrom(ctx context.Context, tx pgx.Tx, tableName string, r io.Reader, columns ...string) (int64, error) {
	stmt := fmt.Sprintf(`COPY %s (%s) FROM STDIN WITH (FORMAT text, DELIMITER E'\t', NULL '\N')`,
		tableName, strings.Join(columns, ", "))

	tag, err := tx.Conn().PgConn().CopyFrom(ctx, r, stmt)
	if err != nil {
		return 0, fmt.Errorf("COPY into %s failed: %w", tableName, err)
	}
	return tag.RowsAffected(), nil
}
