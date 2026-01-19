package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "no quotes, multiple tokens",
			input: "foo bar baz",
			want:  []string{"foo", "bar", "baz"},
		},
		{
			name:  "single quoted token with space",
			input: `foo "bar baz"`,
			// The function retains double quotes, so the second token will contain its quotes.
			want: []string{"foo", `"bar baz"`},
		},
		{
			name:  "multiple quoted tokens",
			input: `"foo bar" "baz qux"`,
			want:  []string{`"foo bar"`, `"baz qux"`},
		},
		{
			name: "leading and trailing spaces",
			// Some leading/trailing spaces, plus a quoted token in the middle.
			input: `  hello   "quoted text"  world  `,
			want:  []string{"hello", `"quoted text"`, "world"},
		},
		{
			name:  "adjacent quoted and unquoted tokens",
			input: `foo"bar" baz`,
			// Because the function writes out quotes verbatim and doesn't handle escaping,
			// we end up toggling inQuotes on the first quote.
			// The exact result depends on how you want to handle such edge cases.
			// With the naive approach, this becomes ["foo\"bar\"", "baz"].
			want: []string{`foo"bar"`, "baz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, split(tt.input))
		})
	}
}
