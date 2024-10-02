package codec

import "encoding/json"

// Convert any data to JSON bytes using generics
func ToJSONBytes[T any](data T) ([]byte, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

// Convert JSON bytes back to Go data using generics
func FromJSONBytes[T any](jsonBytes []byte) (T, error) {
	var result T
	err := json.Unmarshal(jsonBytes, &result)
	return result, err
}
