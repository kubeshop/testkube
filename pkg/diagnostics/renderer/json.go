package renderer

import (
	"encoding/json"
	"os"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

var _ Renderer = JSONRenderer{}

func NewJSONRenderer() JSONRenderer {
	return JSONRenderer{
		encoder: json.NewEncoder(os.Stdout),
	}
}

type JSONRenderer struct {
	encoder *json.Encoder
}

func (r JSONRenderer) RenderGroupStart(message string) {
}

func (r JSONRenderer) RenderProgress(message string) {
}

func (r JSONRenderer) RenderResult(res validators.ValidationResult) {
	r.encoder.Encode(res)
}
