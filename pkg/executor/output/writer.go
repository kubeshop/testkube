package output

import (
	"encoding/json"
	"io"

	"github.com/kubeshop/testkube/pkg/executor/secret"
)

// NewJSONWrapWriter returns new NewJSONWrapWriter instance
func NewJSONWrapWriter(writer io.Writer, envMngr secret.EnvManager) *JSONWrapWriter {
	return &JSONWrapWriter{
		encoder:    json.NewEncoder(writer),
		envManager: envMngr,
	}
}

// JSONWrapWriter wraps bytes stream into json Output of type line
type JSONWrapWriter struct {
	encoder    *json.Encoder
	envManager secret.EnvManager
}

// Write io.Writer method implementation
func (w *JSONWrapWriter) Write(p []byte) (int, error) {
	return len(p), w.encoder.Encode(NewOutputLine(w.envManager.Obfuscate(p)))
}
