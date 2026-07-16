package formatters

import "fmt"

// listCredentialsResponse mirrors the credentials list API response.
// It intentionally reads only non-secret metadata fields — never `value` — so a
// credential value can never reach the model, even if an upstream projection
// starts including one.
type listCredentialsResponse struct {
	Elements []credentialElement `json:"elements"`
}

// credentialElement reads only the reference and scope metadata of a credential.
// The `value` field from the API response is deliberately not declared here.
type credentialElement struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	Reference       string `json:"reference"`
	EnvironmentId   string `json:"environmentId"`
	ResourceGroupId string `json:"resourceGroupId"`
	WorkflowName    string `json:"workflowName"`
}

// formattedCredential is the agent-facing projection of a credential reference.
type formattedCredential struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Scope      string `json:"scope"`
	Reference  string `json:"reference"`
	Expression string `json:"expression"`
}

type formattedCredentialsResult struct {
	Credentials []formattedCredential `json:"credentials"`
}

// FormatListCredentials parses a raw credentials list response (JSON or YAML) and
// returns a compact JSON projection of credential references only — name, type,
// scope, the reference string, and the ready-to-use credential(...) expression.
// It never includes credential values, and it omits non-referenceable
// (execution-scoped) credentials, which the API returns with an empty reference.
func FormatListCredentials(raw string) (string, error) {
	response, isEmpty, err := ParseJSON[listCredentialsResponse](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return `{"credentials":[]}`, nil
	}

	formatted := formattedCredentialsResult{
		Credentials: make([]formattedCredential, 0, len(response.Elements)),
	}

	for _, c := range response.Elements {
		// Skip non-referenceable credentials: execution-scoped credentials carry an
		// empty reference, and the agent can only use referenceable ones in YAML.
		if c.Reference == "" {
			continue
		}
		formatted.Credentials = append(formatted.Credentials, formattedCredential{
			Name:       c.Name,
			Type:       c.Type,
			Scope:      credentialScope(c),
			Reference:  c.Reference,
			Expression: fmt.Sprintf("credential(%q)", c.Reference),
		})
	}

	return FormatJSON(formatted)
}

// credentialScope reports the narrowest scope a credential is bound at, derived
// from which scope identifiers are populated. With the org+environment listing
// this resolves to "organization" or "environment"; the deeper tiers are handled
// for completeness.
func credentialScope(c credentialElement) string {
	switch {
	case c.WorkflowName != "":
		return "workflow"
	case c.ResourceGroupId != "":
		return "resource_group"
	case c.EnvironmentId != "":
		return "environment"
	default:
		return "organization"
	}
}
