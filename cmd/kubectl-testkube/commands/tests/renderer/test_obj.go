package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestRenderer(ui *ui.UI, obj interface{}) error {
	test, ok := obj.(testkube.Test)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.Test in RenderObj for test", obj)
	}

	ui.Warn("Name:     ", test.Name)
	ui.Warn("Namespace:", test.Name)
	ui.Warn("Created:  ", test.Created.String())
	ui.Warn("Labels:   ", testkube.LabelsToString(test.Labels))
	ui.Warn("Schedule: ", test.Schedule)

	if len(test.Params) > 0 {
		ui.Warn("Params: ")
		for k, v := range test.Params {
			ui.Info(k, v)
		}
	}

	if test.Content != nil {
		ui.NL()
		ui.Info("Content")
		ui.Warn("Type", test.Content.Type_)
		if test.Content.Uri != "" {
			ui.Warn("Uri: ", test.Content.Uri)
		}

		if test.Content.Repository != nil {
			ui.Warn("Repository: ")
			ui.Warn("  Uri:      ", test.Content.Repository.Uri)
			ui.Warn("  Branch:   ", test.Content.Repository.Branch)
			ui.Warn("  Path:     ", test.Content.Repository.Path)
			ui.Warn("  Username: ", test.Content.Repository.Username)
			ui.Warn("  Token:    ", test.Content.Repository.Token)
		}

		if test.Content.Data != "" {
			ui.Warn("Data: ", "\n", test.Content.Data)
		}
	}

	return nil

}
