package output

import (
	"encoding/json"
	"io"
)

// NewJSONWrapWriter returns new NewJSONWrapWriter instance
func NewJSONWrapWriter(writer io.Writer) *JSONWrapWriter {
	return &JSONWrapWriter{
		encoder: json.NewEncoder(writer),
	}
}

// JSONWrapWriter wraps bytes stream into json Output of type line
type JSONWrapWriter struct {
	encoder *json.Encoder
}

// Write io.Writer method implementation
func (w *JSONWrapWriter) Write(p []byte) (int, error) {
	return len(p), w.encoder.Encode(NewOutputLine(p))
}
