package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptsAPI(t *testing.T) {

	t.Run("Get executed script details", func(t *testing.T) {
		t.Skip("Implement real test") // TODO implement me
		client := NewDefaultDirectScriptsAPI()
		execution, err := client.GetExecution("test", "60f807e2cd42fbe1286ecfa3")

		fmt.Printf("%+v\n", execution)

		assert.NotNil(t, execution)
		assert.NoError(t, err)

		t.Fail()
	})

	t.Run("Execute script with given ID", func(t *testing.T) {
		t.Skip("Implement real test") // TODO Implement me
		client := NewDefaultDirectScriptsAPI()
		execution, err := client.ExecuteScript("test", "default", "some name", map[string]string{})

		fmt.Printf("%+v\n", execution)

		assert.NotNil(t, execution)
		assert.NoError(t, err)

		t.Fail()
	})

}
