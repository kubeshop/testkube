package crd

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"text/template"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
	"k8s.io/kube-openapi/pkg/validation/validate"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:embed templates
var tf embed.FS

// Template is crd template type
type Template string

const (
	// TemplateExecutor is executor crd template
	TemplateExecutor Template = "executor"
	// TemplateWebhook is webhook crd template
	TemplateWebhook Template = "webhook"
	// TemplateWebhookTemplate is webhook template crd template
	TemplateWebhookTemplate Template = "webhooktemplate"
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
		testkube.WebhookTemplate |
		testkube.TestUpsertRequest |
		testkube.TestSuiteUpsertRequest |
		testkube.ExecutorUpsertRequest |
		testkube.WebhookCreateRequest |
		testkube.WebhookTemplateCreateRequest |
		testkube.TestTrigger |
		testkube.TestTriggerUpsertRequest |
		testkube.TestSource |
		testkube.TestSourceUpsertRequest |
		testkube.Template |
		testkube.TemplateCreateRequest
}

//go:embed schemas
var sf embed.FS

// Schema is crd schema type
type Schema string

const (
	// SchemaTestWorkflow is test workflow crd schema
	SchemaTestWorkflow Schema = "testworkflows.testkube.io_testworkflows"
	// SchemaTestWorkflowTemplate is test workflow template crd schema
	SchemaTestWorkflowTemplate Schema = "testworkflows.testkube.io_testworkflowtemplates"
)

// ExecuteTemplate executes crd template
func ExecuteTemplate(tmpl Template, data any) (string, error) {
	t, err := template.ParseFS(tf, fmt.Sprintf("templates/%s.tmpl", tmpl))
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

func ValidateYAMLAgainstSchema(name Schema, dataYAML []byte) error {
	// Load CRD YAML
	schemaYAML, err := sf.ReadFile(fmt.Sprintf("schemas/%s.yaml", name))
	if err != nil {
		return err
	}

	crd := apiextv1.CustomResourceDefinition{}
	err = yaml.Unmarshal(schemaYAML, &crd)
	if err != nil {
		return err
	}

	// Get YAML schema from CRD for v1 version
	if len(crd.Spec.Versions) == 0 || crd.Spec.Versions[0].Schema == nil {
		return errors.New("schema not found")
	}

	schemaJSON, err := json.Marshal(crd.Spec.Versions[0].Schema.OpenAPIV3Schema)
	if err != nil {
		return err
	}

	schema := new(spec.Schema)
	if err = json.Unmarshal([]byte(schemaJSON), schema); err != nil {
		return err
	}

	dataJSON, err := yaml.ToJSON(dataYAML)
	if err != nil {
		return err
	}

	data := map[string]interface{}{}
	if err = json.Unmarshal(dataJSON, &data); err != nil {
		return err
	}

	// strfmt.Default is the registry of recognized formats
	if err = validate.AgainstSchema(schema, data, strfmt.Default); err != nil {
		return err
	}

	return nil
}
