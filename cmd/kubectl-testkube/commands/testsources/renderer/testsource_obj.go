package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestSourceRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
	testSource, ok := obj.(testkube.TestSource)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestSource in RenderObj for test source", obj)
	}

	ui.Warn("Name:     ", testSource.Name)
	ui.Warn("Namespace:", testSource.Namespace)

	ui.NL()
	ui.Warn("Type", testSource.Type_)
	if testSource.Uri != "" {
		ui.Warn("Uri: ", testSource.Uri)
	}

	if testSource.Repository != nil {
		ui.Warn("Repository: ")
		ui.Warn("  Uri:         ", testSource.Repository.Uri)
		ui.Warn("  Branch:      ", testSource.Repository.Branch)
		ui.Warn("  Commit:      ", testSource.Repository.Commit)
		ui.Warn("  Path:        ", testSource.Repository.Path)
		if testSource.Repository.UsernameSecret != nil {
			ui.Warn("  Username:    ", fmt.Sprintf("[secret:%s key:%s]", testSource.Repository.UsernameSecret.Name,
				testSource.Repository.UsernameSecret.Key))
		}

		if testSource.Repository.TokenSecret != nil {
			ui.Warn("  Token:       ", fmt.Sprintf("[secret:%s key:%s]", testSource.Repository.TokenSecret.Name,
				testSource.Repository.TokenSecret.Key))
		}

		if testSource.Repository.CertificateSecret != "" {
			ui.Warn("  Certificate: ", testSource.Repository.CertificateSecret)
		}

		ui.Warn("  Working dir: ", testSource.Repository.WorkingDir)
		ui.Warn("  Auth type:   ", testSource.Repository.AuthType)
	}

	if testSource.Data != "" {
		ui.Warn("Data: ", "\n", testSource.Data)
	}

	return nil

}
