package marketplace

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtractParameters parses TestWorkflow YAML and returns the entries in
// spec.config as a flat list. It mirrors the Dashboard's extractParameters()
// so the CLI and UI expose the same parameters.
//
// Supports two YAML shapes (both appear in the marketplace):
//
//	spec:
//	  config:
//	    host:
//	      type: string
//	      default: "localhost"
//	      description: "Host to connect to"
//
//	spec:
//	  config:
//	    host: "localhost"
func ExtractParameters(yamlContent []byte) ([]Parameter, error) {
	config, err := findConfigNode(yamlContent)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return parseConfigNode(config), nil
}

// ApplyParameters writes parameter Values back into the YAML, updating each
// key's default field (matching the Dashboard's applyParameters()). Scalar
// config entries are replaced with the value as a string. Unknown
// parameters (not present in spec.config) are silently ignored so the YAML
// is never corrupted by stray --set flags.
//
// As in the Dashboard, when a parameter is marked sensitive but has no
// value, the sensitive flag is dropped to avoid the backend's "value cannot
// be empty" error on credential creation.
func ApplyParameters(yamlContent []byte, params []Parameter) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(yamlContent, &doc); err != nil {
		return nil, fmt.Errorf("parsing workflow YAML: %w", err)
	}
	config := findMappingChild(findMappingChild(rootMapping(&doc), "spec"), "config")
	if config == nil {
		return yamlContent, nil
	}

	for _, param := range params {
		entry := mappingValue(config, param.Key)
		if entry == nil {
			continue
		}
		switch entry.Kind {
		case yaml.MappingNode:
			setMappingString(entry, "default", param.Value)
			if param.Sensitive {
				if param.Value != "" {
					setMappingBool(entry, "sensitive", true)
				} else {
					deleteMappingKey(entry, "sensitive")
				}
			}
		case yaml.ScalarNode:
			entry.Kind = yaml.ScalarNode
			entry.Style = yaml.DoubleQuotedStyle
			entry.Tag = "!!str"
			entry.Value = param.Value
		}
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("re-encoding workflow YAML: %w", err)
	}
	return out, nil
}

// ParseSetFlags converts `key=value` strings (from --set) into updates on
// the provided parameter list, matching by key (case-sensitive, consistent
// with YAML). Unknown keys cause an error so users aren't silently ignored
// when they misspell a parameter name. The returned slice is a copy.
func ParseSetFlags(params []Parameter, overrides []string) ([]Parameter, error) {
	byKey := make(map[string]int, len(params))
	out := make([]Parameter, len(params))
	copy(out, params)
	for i, p := range out {
		byKey[p.Key] = i
	}

	for _, raw := range overrides {
		eq := strings.IndexByte(raw, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("invalid --set value %q: expected key=value", raw)
		}
		key := strings.TrimSpace(raw[:eq])
		val := raw[eq+1:]
		idx, ok := byKey[key]
		if !ok {
			known := make([]string, 0, len(out))
			for _, p := range out {
				known = append(known, p.Key)
			}
			return nil, fmt.Errorf("unknown parameter %q (known: %s)", key, strings.Join(known, ", "))
		}
		out[idx].Value = val
	}
	return out, nil
}

func findConfigNode(yamlContent []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(yamlContent, &doc); err != nil {
		return nil, fmt.Errorf("parsing workflow YAML: %w", err)
	}
	return findMappingChild(findMappingChild(rootMapping(&doc), "spec"), "config"), nil
}

func parseConfigNode(config *yaml.Node) []Parameter {
	params := make([]Parameter, 0, len(config.Content)/2)
	for i := 0; i+1 < len(config.Content); i += 2 {
		keyNode := config.Content[i]
		valNode := config.Content[i+1]
		if keyNode.Kind != yaml.ScalarNode {
			continue
		}

		p := Parameter{Key: keyNode.Value}
		switch valNode.Kind {
		case yaml.MappingNode:
			if def := mappingValue(valNode, "default"); def != nil && def.Kind == yaml.ScalarNode {
				p.Default = def.Value
			}
			if t := mappingValue(valNode, "type"); t != nil && t.Kind == yaml.ScalarNode {
				p.Type = t.Value
			}
			if d := mappingValue(valNode, "description"); d != nil && d.Kind == yaml.ScalarNode {
				p.Description = d.Value
			}
			if s := mappingValue(valNode, "sensitive"); s != nil && s.Kind == yaml.ScalarNode {
				p.Sensitive = s.Value == "true"
			}
		case yaml.ScalarNode:
			p.Default = valNode.Value
		}
		p.Value = p.Default
		params = append(params, p)
	}
	return params
}

func rootMapping(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

func findMappingChild(parent *yaml.Node, key string) *yaml.Node {
	if parent == nil || parent.Kind != yaml.MappingNode {
		return nil
	}
	v := mappingValue(parent, key)
	if v == nil || v.Kind != yaml.MappingNode {
		return nil
	}
	return v
}

func mappingValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Kind == yaml.ScalarNode && m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func setMappingString(m *yaml.Node, key, value string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Kind == yaml.ScalarNode && m.Content[i].Value == key {
			m.Content[i+1].Kind = yaml.ScalarNode
			m.Content[i+1].Style = yaml.DoubleQuotedStyle
			m.Content[i+1].Tag = "!!str"
			m.Content[i+1].Value = value
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Tag: "!!str", Value: value},
	)
}

func setMappingBool(m *yaml.Node, key string, value bool) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	strVal := "false"
	if value {
		strVal = "true"
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Kind == yaml.ScalarNode && m.Content[i].Value == key {
			m.Content[i+1].Kind = yaml.ScalarNode
			m.Content[i+1].Tag = "!!bool"
			m.Content[i+1].Value = strVal
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strVal},
	)
}

func deleteMappingKey(m *yaml.Node, key string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Kind == yaml.ScalarNode && m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}
