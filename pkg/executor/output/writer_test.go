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

		writer := NewJSONWrapWriter(buff)
		line1 := "some log line"
		writer.Write([]byte(line1))
		line2 := "second log line"
		writer.Write([]byte(line2))
		line3 := "second log line"
		writer.Write([]byte(line3))

		lines := bytes.Split(buff.Bytes(), []byte("\n"))

		var output Output
		json.Unmarshal(lines[0], &output)
		assert.Equal(t, line1, output.Content)
		assert.Equal(t, TypeLogLine, output.Type_)

		json.Unmarshal(lines[1], &output)
		assert.Equal(t, line2, output.Content)
		assert.Equal(t, TypeLogLine, output.Type_)

		json.Unmarshal(lines[2], &output)
		assert.Equal(t, line3, output.Content)
		assert.Equal(t, TypeLogLine, output.Type_)

	})

}
