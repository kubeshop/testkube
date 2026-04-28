package testkube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"
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

func (b TemplatableBoxedInteger) MarshalJSON() ([]byte, error) {
	if value, err := strconv.ParseInt(b.Value, 10, 64); err == nil {
		return []byte(`{"value":` + strconv.FormatInt(value, 10) + `}`), nil
	}

	valueJSON, err := json.Marshal(b.Value)
	if err != nil {
		return nil, err
	}

	return append([]byte(`{"value":`), append(valueJSON, '}')...), nil
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

func (b *TemplatableBoxedInteger) UnmarshalBSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var payload templatableBoxedIntegerPayload
	if err := bson.Unmarshal(data, &payload); err != nil {
		return err
	}

	value, err := stringifyTemplatableBoxedInteger(payload.Value)
	if err != nil {
		return err
	}

	b.Value = value
	return nil
}

func (b *TemplatableBoxedInteger) UnmarshalBSONValue(typ byte, data []byte) error {
	raw := bson.RawValue{Type: bson.Type(typ), Value: data}

	switch raw.Type {
	case bson.TypeNull:
		return nil
	case bson.TypeEmbeddedDocument:
		return b.UnmarshalBSON(data)
	case bson.TypeString:
		b.Value = raw.StringValue()
		return nil
	case bson.TypeInt32:
		b.Value = strconv.FormatInt(int64(raw.Int32()), 10)
		return nil
	case bson.TypeInt64:
		b.Value = strconv.FormatInt(raw.Int64(), 10)
		return nil
	case bson.TypeDouble:
		value := raw.Double()
		if float64(int64(value)) != value {
			return fmt.Errorf("templatable boxed integer value must be a whole number")
		}
		b.Value = strconv.FormatInt(int64(value), 10)
		return nil
	default:
		var value interface{}
		if err := raw.Unmarshal(&value); err != nil {
			return err
		}

		stringValue, err := stringifyTemplatableBoxedInteger(value)
		if err != nil {
			return err
		}
		b.Value = stringValue
		return nil
	}
}
