package expressions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToFloat_Error(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "an empty value is quoted", value: "", want: `error while converting value to number: "": invalid syntax`},
		{name: "a number out of range", value: "1e999", want: `error while converting value to number: "1e999": value out of range`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := toFloat(tt.value)
			assert.EqualError(t, err, tt.want)
		})
	}
}
