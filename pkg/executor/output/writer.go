package output

import (
	"encoding/json"
	"io"

	"github.com/kubeshop/testkube/pkg/executor/env"
)

// NewJSONWrapWriter returns new NewJSONWrapWriter instance
func NewJSONWrapWriter(writer io.Writer, envMngr env.Interface) *JSONWrapWriter {
	return &JSONWrapWriter{
		encoder:    json.NewEncoder(writer),
		envManager: envMngr,
	}
}

// JSONWrapWriter wraps bytes stream into json Output of type line
type JSONWrapWriter struct {
	encoder    *json.Encoder
	envManager env.Interface
}

// Write io.Writer method implementation
func (w *JSONWrapWriter) Write(p []byte) (int, error) {
	if w.envManager != nil {
		p = w.envManager.ObfuscateSecrets(p)
	}
	return len(p), w.encoder.Encode(NewOutputLine(p))
}
