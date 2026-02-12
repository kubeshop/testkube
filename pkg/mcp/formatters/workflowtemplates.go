package formatters

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// formattedWorkflowTemplate is a compact representation of a workflow template for MCP responses.
type formattedWorkflowTemplate struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Created     time.Time         `json:"created,omitempty"`
	Updated     time.Time         `json:"updated,omitempty"`
}

// FormatListWorkflowTemplates parses a raw API response (JSON) containing
// []testkube.TestWorkflowTemplate and returns a compact JSON
// representation with only essential fields.
func FormatListWorkflowTemplates(raw string) (string, error) {
	templates, isEmpty, err := ParseJSON[[]testkube.TestWorkflowTemplate](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "[]", nil
	}

	formatted := make([]formattedWorkflowTemplate, 0, len(templates))
	for _, tpl := range templates {
		f := formattedWorkflowTemplate{
			Name:        tpl.Name,
			Namespace:   tpl.Namespace,
			Description: tpl.Description,
			Labels:      tpl.Labels,
			Created:     tpl.Created,
			Updated:     tpl.Updated,
		}
		formatted = append(formatted, f)
	}

	return FormatJSON(formatted)
}

// FormatGetWorkflowTemplateDefinition is a pass-through formatter for workflow template definitions.
// The full YAML/JSON definition is needed by AI to understand the complete template
// structure including all steps, configuration schema, and templates.
// No fields are stripped since AI needs the complete spec for analysis and modification.
func FormatGetWorkflowTemplateDefinition(raw string) (string, error) {
	// Validate that input is parseable, but return as-is
	_, isEmpty, err := ParseJSON[testkube.TestWorkflowTemplate](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}
	// Return unchanged - AI needs full definition for template analysis
	return raw, nil
}

// formattedCreateWorkflowTemplateResult is a compact representation of template creation/update result.
type formattedCreateWorkflowTemplateResult struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Created     time.Time         `json:"created,omitempty"`
}

// FormatCreateWorkflowTemplate parses a raw API response containing the created workflow template.
// Returns a compact JSON confirming the template was created with key metadata.
// Strips: full spec (already known by caller), annotations, status.
func FormatCreateWorkflowTemplate(raw string) (string, error) {
	tpl, isEmpty, err := ParseJSON[testkube.TestWorkflowTemplate](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	formatted := formattedCreateWorkflowTemplateResult{
		Name:        tpl.Name,
		Namespace:   tpl.Namespace,
		Description: tpl.Description,
		Labels:      tpl.Labels,
		Created:     tpl.Created,
	}

	return FormatJSON(formatted)
}

// FormatUpdateWorkflowTemplate parses a raw API response containing the updated workflow template.
// Uses the same compact format as FormatCreateWorkflowTemplate.
// Strips: full spec (already known by caller), annotations, status.
func FormatUpdateWorkflowTemplate(raw string) (string, error) {
	return FormatCreateWorkflowTemplate(raw)
}
