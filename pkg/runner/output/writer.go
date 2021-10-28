// TODO will be moved to testkube lib when completed
package output

import (
	"encoding/json"
	"io"
)

func NewJSONWrapWriter(writer io.Writer) *JSONWrapWriter {
	return &JSONWrapWriter{
		writer: writer,
	}
}

type JSONWrapWriter struct {
	writer io.Writer
}

func (e JSONWrapWriter) Write(p []byte) (int, error) {
	output, err := json.Marshal(NewOutputLine(p))
	if err != nil {
		return 0, err
	}

	return e.writer.Write(output)
}
