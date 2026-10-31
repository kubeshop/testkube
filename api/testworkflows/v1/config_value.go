package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ConfigValue accepts an integer or a string and keeps the raw scalar
// text of the value. A type that stores numbers as int32, such as
// intstr.IntOrString, fails to unmarshal a larger number. One bad value
// then causes a full informer list to fail.

// +kubebuilder:validation:XIntOrString
type ConfigValue string

func NewConfigValue(value string) *ConfigValue {
	result := ConfigValue(value)
	return &result
}

func (v ConfigValue) String() string {
	return string(v)
}

func (ConfigValue) OpenAPISchemaType() []string {
	return []string{"string"}
}

func (ConfigValue) OpenAPISchemaFormat() string {
	return "int-or-string"
}

func (ConfigValue) OpenAPIV3OneOfTypes() []string {
	return []string{"integer", "string"}
}

func (v ConfigValue) MarshalJSON() ([]byte, error) {
	value := v.String()
	// Emit a number only when the value is a canonical integer in the
	// int32 range. Older clients decode this field with
	// intstr.IntOrString, which stores numbers as int32. For a larger
	// number, the string form prevents a decode error in those clients.
	if parsed, err := strconv.ParseInt(value, 10, 32); err == nil && strconv.FormatInt(parsed, 10) == value {
		return []byte(value), nil
	}
	return json.Marshal(value)
}

func (v *ConfigValue) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	// json.Number keeps the raw literal, so numbers of any size keep
	// their full precision.
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}

	switch value := value.(type) {
	case string:
		*v = ConfigValue(value)
	case json.Number:
		*v = ConfigValue(value.String())
	case bool:
		*v = ConfigValue(strconv.FormatBool(value))
	default:
		return fmt.Errorf("configuration value must be a scalar")
	}
	return nil
}

func (v *ConfigValue) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("configuration value must be a scalar")
	}
	if node.Tag == "!!null" {
		return nil
	}
	*v = ConfigValue(node.Value)
	return nil
}
