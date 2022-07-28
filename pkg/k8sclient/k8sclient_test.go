//go:build integration

package k8sclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClusterVersion(t *testing.T) {
	client, err := ConnectToK8s()
	assert.NoError(t, err)

	v, err := GetClusterVersion(client)
	assert.NoError(t, err)
	assert.NotEmpty(t, v)
}

func TestGetAPIServerLogs(t *testing.T) {
	client, err := ConnectToK8s()
	assert.NoError(t, err)

	logs, err := GetAPIServerLogs(context.Background(), client, "testkube")
	assert.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestGetOperatorLogs(t *testing.T) {
	client, err := ConnectToK8s()
	assert.NoError(t, err)

	logs, err := GetOperatorLogs(context.Background(), client, "testkube")
	assert.NoError(t, err)
	assert.NotEmpty(t, logs)
}
