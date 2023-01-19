package renderer

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestRenderer(ui *ui.UI, obj interface{}) error {
	test, ok := obj.(testkube.Test)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.Test in RenderObj for test", obj)
	}

	ui.Warn("Name:     ", test.Name)
	ui.Warn("Namespace:", test.Namespace)
	ui.Warn("Created:  ", test.Created.String())
	if len(test.Labels) > 0 {
		ui.NL()
		ui.Warn("Labels:   ", testkube.MapToString(test.Labels))
	}
	if test.Schedule != "" {
		ui.NL()
		ui.Warn("Schedule: ", test.Schedule)
	}

	if test.Content != nil {
		ui.NL()
		ui.Info("Content")
		ui.Warn("Type", test.Content.Type_)
		if test.Content.Uri != "" {
			ui.Warn("Uri: ", test.Content.Uri)
		}

		if test.Content.Repository != nil {
			ui.Warn("Repository:    ")
			ui.Warn("  Uri:         ", test.Content.Repository.Uri)
			ui.Warn("  Branch:      ", test.Content.Repository.Branch)
			ui.Warn("  Commit:      ", test.Content.Repository.Commit)
			ui.Warn("  Path:        ", test.Content.Repository.Path)
			if test.Content.Repository.UsernameSecret != nil {
				ui.Warn("  Username:    ", fmt.Sprintf("[secret:%s key:%s]", test.Content.Repository.UsernameSecret.Name,
					test.Content.Repository.UsernameSecret.Key))
			}

			if test.Content.Repository.TokenSecret != nil {
				ui.Warn("  Token:       ", fmt.Sprintf("[secret:%s key:%s]", test.Content.Repository.TokenSecret.Name,
					test.Content.Repository.TokenSecret.Key))
			}

			if test.Content.Repository.CertificateSecret != "" {
				ui.Warn("  Certificate: ", test.Content.Repository.CertificateSecret)
			}

			ui.Warn("  Working dir: ", test.Content.Repository.WorkingDir)
		}

		if test.Content.Data != "" {
			ui.Warn("Data: ", "\n", test.Content.Data)
		}
	}

	if test.Source != "" {
		ui.NL()
		ui.Warn("Source: ", test.Source)
	}

	if test.ExecutionRequest != nil {
		ui.Warn("Execution request: ")
		if test.ExecutionRequest.Name != "" {
			ui.Warn("  Name:                 ", test.ExecutionRequest.Name)
		}

		if len(test.ExecutionRequest.Variables) > 0 {
			renderer.RenderVariables(test.ExecutionRequest.Variables)
		}

		if len(test.ExecutionRequest.Args) > 0 {
			ui.Warn("  Args:                 ", test.ExecutionRequest.Args...)
		}

		if len(test.ExecutionRequest.Envs) > 0 {
			ui.NL()
			ui.Warn("  Envs:                 ", testkube.MapToString(test.ExecutionRequest.Envs))
		}

		if len(test.ExecutionRequest.SecretEnvs) > 0 {
			ui.NL()
			ui.Warn("  Secret Envs:          ", testkube.MapToString(test.ExecutionRequest.SecretEnvs))
		}

		if test.ExecutionRequest.VariablesFile != "" {
			ui.Warn("  Variables file:       ", "\n", test.ExecutionRequest.VariablesFile)
		}

		if test.ExecutionRequest.HttpProxy != "" {
			ui.Warn("  Http proxy:           ", test.ExecutionRequest.HttpProxy)
		}

		if test.ExecutionRequest.HttpsProxy != "" {
			ui.Warn("  Https proxy:          ", test.ExecutionRequest.HttpsProxy)
		}

		if test.ExecutionRequest.ArtifactRequest != nil {
			ui.Warn("  Artifact request:     ")
			ui.Warn("    Storage class name: ", test.ExecutionRequest.ArtifactRequest.StorageClassName)
			ui.Warn("    Volume mount path:  ", test.ExecutionRequest.ArtifactRequest.VolumeMountPath)
			ui.Warn("    Dirs:               ", strings.Join(test.ExecutionRequest.ArtifactRequest.Dirs, ","))
		}

		if test.ExecutionRequest.JobTemplate != "" {
			ui.Warn("  Job template:         ", "\n", test.ExecutionRequest.JobTemplate)
		}

		if test.ExecutionRequest.PreRunScript != "" {
			ui.Warn("  Pre run script:       ", "\n", test.ExecutionRequest.PreRunScript)
		}

		if test.ExecutionRequest.ScraperTemplate != "" {
			ui.Warn("  Scraper template:     ", "\n", test.ExecutionRequest.ScraperTemplate)
		}
	}

	return nil

}
