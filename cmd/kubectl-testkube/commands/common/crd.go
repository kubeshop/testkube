package common

import (
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

// UIPrintCRD prints crd to ui
func UIPrintCRD(tmpl crd.Template, object any, firstEntry *bool) {
	data, err := crd.ExecuteTemplate(tmpl, object)
	ui.ExitOnError("executing crd template", err)
	if !*firstEntry {
		ui.Info("---")
	} else {
		*firstEntry = false
	}
	ui.Info(data)
}
