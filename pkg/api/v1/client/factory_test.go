package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClient(t *testing.T) {

	t.Run("get proxy client", func(t *testing.T) {
		t.Skip("This one needs kubernetes config to work")

		client, err := GetClient(ClientProxy, "testkube")
		assert.NoError(t, err)
		assert.Equal(t, "client.ProxyAPIClient", fmt.Sprintf("%T", client))
	})

	t.Run("get direct client", func(t *testing.T) {
		client, err := GetClient(ClientDirect, "testkube")
		assert.NoError(t, err)
		assert.Equal(t, "client.DirectAPIClient", fmt.Sprintf("%T", client))
	})
}
