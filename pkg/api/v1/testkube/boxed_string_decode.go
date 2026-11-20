package testkube

import (
	"bytes"
	"encoding/json"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// UnmarshalJSON implements json.Unmarshaler for BoxedString.
// It accepts the object form and a bare number, because a field such as
// parallel.parallelism was an integer before, and stored records still hold it.
func (b *BoxedString) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	if trimmed[0] != '{' {
		var number json.Number
		if err := json.Unmarshal(trimmed, &number); err == nil {
			b.Value = number.String()
			return nil
		}
	}

	type boxedStringAlias BoxedString
	var alias boxedStringAlias
	if err := json.Unmarshal(trimmed, &alias); err != nil {
		return err
	}
	*b = BoxedString(alias)
	return nil
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler for BoxedString.
// It accepts the embedded document and a bare number, for the same reason as UnmarshalJSON.
func (b *BoxedString) UnmarshalBSONValue(typ byte, data []byte) error {
	raw := bson.RawValue{Type: bson.Type(typ), Value: data}

	switch raw.Type {
	case bson.TypeNull, bson.TypeUndefined:
		return nil
	case bson.TypeInt32:
		b.Value = strconv.FormatInt(int64(raw.Int32()), 10)
		return nil
	case bson.TypeInt64:
		b.Value = strconv.FormatInt(raw.Int64(), 10)
		return nil
	case bson.TypeDouble:
		b.Value = strconv.FormatFloat(raw.Double(), 'f', -1, 64)
		return nil
	default:
		type boxedStringAlias BoxedString
		var alias boxedStringAlias
		if err := raw.Unmarshal(&alias); err != nil {
			return err
		}
		*b = BoxedString(alias)
		return nil
	}
}
