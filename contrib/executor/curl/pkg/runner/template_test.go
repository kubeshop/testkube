package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveTemplateSuccess(t *testing.T) {
	assertion := require.New(t)
	template := "value1 {{.value1}}, value2 {{.value2}}"
	params := map[string]string{"value1": "1", "value2": "2"}

	result, err := ResolveTemplate(template, params)
	assertion.NoError(err)
	assertion.Equal("value1 1, value2 2", result)
}

func TestResolveTemplatesSuccess(t *testing.T) {
	assertion := require.New(t)
	template1 := "value1 {{.value1}}, value2 {{.value2}}"
	template2 := "value3 {{.value3}}, value4 {{.value4}}"
	template3 := "unchanged"
	params := map[string]string{"value1": "1", "value2": "2", "value3": "3", "value4": "4"}
	templates := []string{template1, template2, template3}

	err := ResolveTemplates(templates, params)
	assertion.NoError(err)
	assertion.Equal("value1 1, value2 2", templates[0])
	assertion.Equal("value3 3, value4 4", templates[1])
	assertion.Equal(template3, templates[2])
}
