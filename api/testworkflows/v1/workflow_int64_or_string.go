package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// WorkflowInt64OrString stores Kubernetes security-context IDs as either an
// int64-compatible numeric value or a Testkube template string.
type WorkflowInt64OrString string

func NewWorkflowInt64OrString(value string) *WorkflowInt64OrString {
	result := WorkflowInt64OrString(value)
	return &result
}

func (v WorkflowInt64OrString) String() string {
	return string(v)
}

func (WorkflowInt64OrString) OpenAPISchemaType() []string {
	return []string{"string"}
}

func (WorkflowInt64OrString) OpenAPISchemaFormat() string {
	return "int-or-string"
}

func (WorkflowInt64OrString) OpenAPIV3OneOfTypes() []string {
	return []string{"integer", "string"}
}

func (v WorkflowInt64OrString) MarshalJSON() ([]byte, error) {
	value := v.String()
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return []byte(value), nil
	}
	return json.Marshal(value)
}

func (v *WorkflowInt64OrString) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		*v = WorkflowInt64OrString(value)
		return nil
	}

	var value json.Number
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if _, err := value.Int64(); err != nil {
		return err
	}
	*v = WorkflowInt64OrString(value.String())
	return nil
}

func (v *WorkflowInt64OrString) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("workflow int64-or-string value must be a scalar")
	}
	if node.Tag == "!!int" {
		var value int64
		if err := node.Decode(&value); err != nil {
			return err
		}
		*v = WorkflowInt64OrString(strconv.FormatInt(value, 10))
		return nil
	}
	*v = WorkflowInt64OrString(node.Value)
	return nil
}
