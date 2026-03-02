package formatters

// listAgentsResponse mirrors the API response structure for list agents.
type listAgentsResponse struct {
	Elements []agentElement `json:"elements"`
}

// agentElement represents an agent from the raw API response.
type agentElement struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version,omitempty"`
	Namespace    string            `json:"namespace,omitempty"`
	Disabled     bool              `json:"disabled,omitempty"`
	Floating     bool              `json:"floating,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	IsSuperAgent bool              `json:"isSuperAgent,omitempty"`
}

// formattedAgentsResult is a compact representation of agent list results.
type formattedAgentsResult struct {
	Elements []formattedAgent `json:"elements"`
}

// formattedAgent is a compact representation of an agent for MCP responses.
type formattedAgent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Namespace    string   `json:"namespace,omitempty"`
	Disabled     bool     `json:"disabled,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	IsSuperAgent bool     `json:"isSuperAgent,omitempty"`
}

// FormatListAgents parses a raw API response (JSON or YAML) containing
// the list agents response and returns a compact JSON representation
// with only essential fields.
// It strips environments, accessedAt, createdAt, labels, and runnerPolicy.
func FormatListAgents(raw string) (string, error) {
	response, isEmpty, err := ParseJSON[listAgentsResponse](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return `{"elements":[]}`, nil
	}

	formatted := formattedAgentsResult{
		Elements: make([]formattedAgent, 0, len(response.Elements)),
	}

	for _, agent := range response.Elements {
		f := formattedAgent{
			ID:           agent.ID,
			Name:         agent.Name,
			Version:      agent.Version,
			Namespace:    agent.Namespace,
			Disabled:     agent.Disabled,
			Capabilities: agent.Capabilities,
			IsSuperAgent: agent.IsSuperAgent,
		}

		formatted.Elements = append(formatted.Elements, f)
	}

	return FormatJSON(formatted)
}
