package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/executor/env"
)

func TestJSONWrapWritter(t *testing.T) {

	t.Run("test if output is wrapped correctly", func(t *testing.T) {

		buff := bytes.NewBuffer([]byte(""))

		writer := NewJSONWrapWriter(buff, env.NewManager())
		line1 := "some log line"
		_, err := writer.Write([]byte(line1))
		assert.NoError(t, err)
		line2 := "second log line"
		_, err = writer.Write([]byte(line2))
		assert.NoError(t, err)
		line3 := "second log line"
		_, err = writer.Write([]byte(line3))
		assert.NoError(t, err)
		lines := bytes.Split(buff.Bytes(), []byte("\n"))

		var output Output
		err = json.Unmarshal(lines[0], &output)
		assert.NoError(t, err)
		assert.Equal(t, line1, output.Content)
		assert.Equal(t, TypeLogLine, output.Type_)

		err = json.Unmarshal(lines[1], &output)
		assert.NoError(t, err)
		assert.Equal(t, line2, output.Content)
		assert.Equal(t, TypeLogLine, output.Type_)

		err = json.Unmarshal(lines[2], &output)
		assert.NoError(t, err)
		assert.Equal(t, line3, output.Content)
		assert.Equal(t, TypeLogLine, output.Type_)

	})

}
