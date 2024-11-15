package renderer

import "github.com/kubeshop/testkube/pkg/diagnostics/validators"

type Renderer interface {
	RenderGroupStart(group string)
	RenderResult(validators.ValidationResult)
	RenderProgress(message string)
}
