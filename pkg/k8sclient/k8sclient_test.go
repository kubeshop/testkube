package k8sclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetClusterVersion(t *testing.T) {
	client := fake.NewSimpleClientset()

	v, err := GetClusterVersion(client)
	assert.NoError(t, err)
	assert.Equal(t, "v0.0.0-master+$Format:%H$", v)
}

func TestGetAPIServerLogs(t *testing.T) {
	client := fake.NewSimpleClientset()

	logs, err := GetAPIServerLogs(context.Background(), client, "testkube")
	assert.NoError(t, err)
	assert.Equal(t, []string([]string{}), logs)
}

func TestGetOperatorLogs(t *testing.T) {
	client := fake.NewSimpleClientset()

	logs, err := GetOperatorLogs(context.Background(), client, "testkube")
	assert.NoError(t, err)
	assert.Equal(t, []string([]string{}), logs)
}

func TestGetPodLogs(t *testing.T) {
	client := fake.NewSimpleClientset()

	logs, err := GetPodLogs(context.Background(), client, "testkube", "selector")
	assert.NoError(t, err)
	assert.Equal(t, []string([]string{}), logs)
}
