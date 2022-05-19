package renderer

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func RenderStringArray(title string, a []string) {
	if len(a) > 0 {
		ui.Warn(title+":", a...)
	}
}

func RenderStringMap(title string, m map[string]string) {
	if len(m) > 0 {
		ui.Warn(title+":", testkube.MapToString(m))
	}
}
