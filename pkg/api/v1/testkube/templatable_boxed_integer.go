package testkube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

type templatableBoxedIntegerPayload struct {
	Value interface{} `json:"value" yaml:"value"`
}

func stringifyTemplatableBoxedInteger(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		if float64(int64(v)) != v {
			return "", fmt.Errorf("templatable boxed integer value must be a whole number")
		}
		return strconv.FormatInt(int64(v), 10), nil
	case json.Number:
		if _, err := v.Int64(); err == nil {
			return v.String(), nil
		}
		return "", fmt.Errorf("templatable boxed integer value must be a whole number")
	default:
		return "", fmt.Errorf("unsupported templatable boxed integer value type %T", value)
	}
}

func (b *TemplatableBoxedInteger) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	if trimmed[0] == '{' {
		var payload templatableBoxedIntegerPayload
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			return err
		}
		value, err := stringifyTemplatableBoxedInteger(payload.Value)
		if err != nil {
			return err
		}
		b.Value = value
		return nil
	}

	var value interface{}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	stringValue, err := stringifyTemplatableBoxedInteger(value)
	if err != nil {
		return err
	}
	b.Value = stringValue
	return nil
}

func (b *TemplatableBoxedInteger) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		var value interface{}
		if err := node.Decode(&value); err != nil {
			return err
		}
		stringValue, err := stringifyTemplatableBoxedInteger(value)
		if err != nil {
			return err
		}
		b.Value = stringValue
		return nil
	}

	type alias TemplatableBoxedInteger
	var payload templatableBoxedIntegerPayload
	if err := node.Decode(&payload); err == nil && payload.Value != nil {
		stringValue, err := stringifyTemplatableBoxedInteger(payload.Value)
		if err != nil {
			return err
		}
		b.Value = stringValue
		return nil
	}

	var decoded alias
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	*b = TemplatableBoxedInteger(decoded)
	return nil
}
