package scalars

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

// Generic implementation

func MarshalAnyMapScalar[T interface{}](val map[string]T) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		err := json.NewEncoder(w).Encode(val)
		if err != nil {
			panic(err)
		}
	})
}

func UnmarshalAnyMapScalar[T interface{}](v interface{}) (map[string]T, error) {
	if m, ok := v.(map[string]T); ok {
		return m, nil
	}
	return nil, fmt.Errorf("%T is not a map[string]%T", v, *new(T))
}

// Specific types

func MarshalStringMapScalar(val map[string]string) graphql.Marshaler {
	return MarshalAnyMapScalar(val)
}

func UnmarshalStringMapScalar(v interface{}) (map[string]string, error) {
	return UnmarshalAnyMapScalar[string](v)
}
