package common

import (
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

// UIPrintCRD prints crd to ui
func UIPrintCRD(tmpl crd.Template, object any, firstEntry *bool) {
	data, err := crd.ExecuteTemplate(tmpl, object)
	ui.ExitOnError("executing crd template", err)
	ui.Info(data)
	if !*firstEntry {
		ui.Info("\n---")
	} else {
		*firstEntry = false
	}
}
