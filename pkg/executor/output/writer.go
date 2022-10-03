package output

import (
	"encoding/json"
	"io"

	"github.com/kubeshop/testkube/pkg/executor/secret"
)

// NewJSONWrapWriter returns new NewJSONWrapWriter instance
func NewJSONWrapWriter(writer io.Writer, envMngr secret.Manager) *JSONWrapWriter {
	return &JSONWrapWriter{
		encoder:    json.NewEncoder(writer),
		envManager: envMngr,
	}
}

// JSONWrapWriter wraps bytes stream into json Output of type line
type JSONWrapWriter struct {
	encoder    *json.Encoder
	envManager secret.Manager
}

// Write io.Writer method implementation
func (w *JSONWrapWriter) Write(p []byte) (int, error) {
	if w.envManager != nil {
		p = w.envManager.Obfuscate(p)
	}
	return len(p), w.encoder.Encode(NewOutputLine(p))
}
