package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPClient(t *testing.T) {

	t.Run("client execute script", func(t *testing.T) {
		t.Skip("Implement valid script") // TODO  implement me

		client := NewHTTPExecutorClient(Config{URI: "http://localhost:8082"})
		e, err := client.Execute(ExecuteOptions{Content: "", Params: make(map[string]string)}) // To be fixed with the proper types call.

		assert.NoError(t, err)
		assert.NotEqual(t, "", e)

		fmt.Printf("%+v\n", e)
		t.Fail()

	})

}
