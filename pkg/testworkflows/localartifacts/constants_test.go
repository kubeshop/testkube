package localartifacts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid string
	}{
		{name: "nested path", value: "reports/./junit.xml", valid: "reports/junit.xml"},
		{name: "empty", value: ""},
		{name: "absolute", value: "/reports/junit.xml"},
		{name: "traversal", value: "../reports/junit.xml"},
		{name: "backslash", value: `reports\junit.xml`},
		{name: "volume separator", value: "C:reports/junit.xml"},
		{name: "too many components", value: "a/b/c/d/e/f/g/h/i"},
		{name: "nul", value: string([]byte{'r', 'e', 'p', 'o', 'r', 't', 's', 0, 'j', 'u', 'n', 'i', 't', '.', 'x', 'm', 'l'})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ValidateRelativePath(test.value)
			if test.valid == "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.valid, actual)
		})
	}
}

func TestValidateStepRef(t *testing.T) {
	actual, err := ValidateStepRef("root-03")
	require.NoError(t, err)
	require.Equal(t, "root-03", actual)

	_, err = ValidateStepRef("root/03")
	require.ErrorContains(t, err, "single path segment")
}
