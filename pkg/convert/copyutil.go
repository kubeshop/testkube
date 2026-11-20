package convert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
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

// formatInt renders an integer for a numeric column.
func formatInt[T int32 | int64](v T) string {
	return strconv.FormatInt(int64(v), 10)
}

// copyField is one value of a COPY row, with the column that it belongs to. The
// value is the final field text: escaped, formatted, or the NULL sentinel.
type copyField struct {
	column string
	value  string
}

// writeCopyRow writes one COPY text row. It fails when the fields do not name
// the columns of the table in the same order. COPY reads fields by position, so
// without this check a moved or missing value lands in a different column.
func writeCopyRow(w io.Writer, columns []string, fields []copyField) error {
	if len(fields) != len(columns) {
		return fmt.Errorf("COPY row has %d fields for %d columns", len(fields), len(columns))
	}
	var row strings.Builder
	for i, field := range fields {
		if field.column != columns[i] {
			return fmt.Errorf("COPY row has the field %q at the position of the column %q", field.column, columns[i])
		}
		if i > 0 {
			row.WriteByte('\t')
		}
		row.WriteString(field.value)
	}
	row.WriteByte('\n')
	_, err := io.WriteString(w, row.String())
	return err
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
