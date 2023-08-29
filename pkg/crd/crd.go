package crd

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:embed templates
var f embed.FS

// Template is crd template type
type Template string

const (
	// TemplateExecutor is executor crd template
	TemplateExecutor Template = "executor"
	// TemplateWebhook is webhook crd template
	TemplateWebhook Template = "webhook"
	// TemplateTest is test crd template
	TemplateTest Template = "test"
	// TemplateTestSuite is test suite crd template
	TemplateTestSuite Template = "testsuite"
	// TemplateTestTrigger is test trigger crd template
	TemplateTestTrigger Template = "testtrigger"
	// TemplateTestSource is test source crd template
	TemplateTestSource Template = "testsource"
	// TemplateTemplate is template crd template
	TemplateTemplate Template = "template"
)

// Gettable is an interface of gettable objects
type Gettable interface {
	testkube.Test |
		testkube.TestSuite |
		testkube.Webhook |
		testkube.TestUpsertRequest |
		testkube.TestSuiteUpsertRequest |
		testkube.ExecutorUpsertRequest |
		testkube.WebhookCreateRequest |
		testkube.TestTrigger |
		testkube.TestTriggerUpsertRequest |
		testkube.TestSource |
		testkube.TestSourceUpsertRequest |
		testkube.Template |
		testkube.TemplateCreateRequest
}

// ExecuteTemplate executes crd template
func ExecuteTemplate(tmpl Template, data any) (string, error) {
	t, err := template.ParseFS(f, fmt.Sprintf("templates/%s.tmpl", tmpl))
	if err != nil {
		return "", err
	}

	buffer := bytes.NewBuffer([]byte{})
	err = t.Execute(buffer, data)
	return buffer.String(), err
}

// GenerateYAML generates CRDs yaml for Testkube models
func GenerateYAML[G Gettable](tmpl Template, items []G) (string, error) {
	data := ""
	firstEntry := true
	for _, item := range items {
		result, err := ExecuteTemplate(tmpl, item)
		if err != nil {
			return "", fmt.Errorf("could not populate YAML template for %s: %w", tmpl, err)
		}

		if !firstEntry {
			data += "\n---\n"
		} else {
			firstEntry = false
		}

		data += result
	}

	return data, nil
}
