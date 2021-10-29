// TODO will be moved to testkube lib when completed
package output

import (
	"encoding/json"
	"io"
)

func NewJSONWrapWriter(writer io.Writer) *JSONWrapWriter {
	return &JSONWrapWriter{
		encoder: json.NewEncoder(writer),
	}
}

type JSONWrapWriter struct {
	encoder *json.Encoder
}

func (w *JSONWrapWriter) Write(p []byte) (int, error) {
	return len(p), w.encoder.Encode(NewOutputLine(p))
}
