package utils

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRoundDuration(t *testing.T) {
	t.Run("round duration should round to default 100ms", func(t *testing.T) {

		//given
		d := time.Duration(99111111111)

		// when
		rounded := RoundDuration(d)

		// then
		assert.Equal(t, "1m39.11s", rounded.String())
	})

	t.Run("round duration should round to given value", func(t *testing.T) {

		//given
		d := time.Duration(99111111111)

		// when
		rounded := RoundDuration(d, time.Minute)

		// then
		assert.Equal(t, "2m0s", rounded.String())
	})
}

func TestSanitizeName(t *testing.T) {

	t.Run("name should not be changed", func(t *testing.T) {
		//given
		name := "abc-123"

		// when
		sanized := SanitizeName(name)

		// then
		assert.Equal(t, "abc-123", sanized)
	})

	t.Run("name should be shorted", func(t *testing.T) {
		//given
		name := "abc" + strings.Repeat("0123456789", 10)

		// when
		sanized := SanitizeName(name)

		// then
		assert.Equal(t, "abc"+strings.Repeat("0123456789", 6), sanized)
	})

	t.Run("name should be sanitized", func(t *testing.T) {
		//given
		name := "@#$%!abc()~+123{}<>;"

		// when
		sanized := SanitizeName(name)

		// then
		assert.Equal(t, "abc-123", sanized)
	})
}

func TestSanitizeLabelValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty stays empty", value: "", want: ""},
		{name: "valid value unchanged", value: "my-workflow.name_1", want: "my-workflow.name_1"},
		{name: "invalid chars replaced with hyphen", value: "my workflow/name", want: "my-workflow-name"},
		{name: "leading and trailing punctuation trimmed", value: ".-my-workflow-.", want: "my-workflow"},
		{name: "truncated to 63 chars", value: strings.Repeat("a", 100), want: strings.Repeat("a", 63)},
		{name: "unsanitizable value becomes empty", value: strings.Repeat("/", 5), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLabelValue(tt.value)
			assert.Equal(t, tt.want, got)
			if got != "" {
				assert.LessOrEqual(t, len(got), 63)
			}
		})
	}
}

func TestNewTemplate(t *testing.T) {

	t.Run("sprig functions should be available", func(t *testing.T) {
		//given
		template := `{{ default "foo" .Bar }}`

		// when
		tpl, err := NewTemplate("test").Parse(template)
		assert.NoError(t, err)
		var result bytes.Buffer
		assert.NoError(t, tpl.Execute(&result, nil))

		// then
		assert.Equal(t, "foo", result.String())
	})
}
