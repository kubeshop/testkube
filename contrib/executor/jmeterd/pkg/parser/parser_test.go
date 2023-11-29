package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseJTLReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantError bool
		notEmpty  bool
	}{
		{
			name:      "errored XML report",
			input:     errorXML,
			wantError: true,
			notEmpty:  false,
		},
		{
			name:      "errored CSV report",
			input:     errorCSV,
			wantError: true,
			notEmpty:  false,
		},
		{
			name:      "success XML report",
			input:     successXML,
			wantError: false,
			notEmpty:  true,
		},
		{
			name:      "success CSV report",
			input:     successCSV,
			wantError: false,
			notEmpty:  true,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := []byte("test completed\n")
			result, err := ParseJTLReport(strings.NewReader(tt.input), output)

			if tt.wantError {
				assert.ErrorIs(t, err, ErrEmptyReport)
			} else {
				assert.NoError(t, err)
			}

			if tt.notEmpty {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}
