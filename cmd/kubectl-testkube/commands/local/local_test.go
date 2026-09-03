package local

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/testworkflows/localrunner"
)

func TestNewLocalCmdUsesAnIndependentLocalNamespaceDefault(t *testing.T) {
	command := NewLocalCmd()
	namespace := command.PersistentFlags().Lookup("namespace")
	require.NotNil(t, namespace)
	assert.Equal(t, localrunner.DefaultNamespace, namespace.DefValue)
	for _, name := range []string{"run", "pause", "resume", "shell", "clean"} {
		found, _, err := command.Find([]string{name})
		require.NoError(t, err)
		assert.Equal(t, name, found.Name())
	}
}

func TestParseAssignmentsAndRunIDValidation(t *testing.T) {
	assignments, err := parseAssignments([]string{"first=value", "empty="}, "--config")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"first": "value", "empty": ""}, assignments)
	for _, values := range [][]string{{"missing"}, {"=value"}, {"same=first", "same=second"}} {
		_, err = parseAssignments(values, "--config")
		require.Error(t, err)
		assert.True(t, localrunner.IsUsageError(err))
	}
	assert.NoError(t, exactlyOneRunID(&cobra.Command{}, []string{"local-safe"}))
	assert.True(t, localrunner.IsUsageError(exactlyOneRunID(&cobra.Command{}, []string{"bad/run"})))
	assert.True(t, localrunner.IsUsageError(exactlyOneRunID(&cobra.Command{}, nil)))
}
