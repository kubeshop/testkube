package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONWrapWritter(t *testing.T) {

	t.Run("test if output is wrapped correctly", func(t *testing.T) {

		buff := bytes.NewBuffer([]byte(""))

		writer := JSONWrapWriter{writer: buff}
		line := "some log line"
		writer.Write([]byte(line))

		var output Output
		json.Unmarshal(buff.Bytes(), &output)

		assert.Equal(t, line, output.Content)
		assert.Equal(t, TypeLogLine, output.Type)
		assert.Equal(t, false, output.Error)
	})

}
