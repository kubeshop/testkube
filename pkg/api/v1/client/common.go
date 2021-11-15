package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/kubeshop/testkube/pkg/runner/output"
)

const Version = "v1"

// Converts io.Reader with SSE data like `data: {"type": "event", "message":"something"}`
// to channel of output.Output objects, helps with logs streaming from SSE endpoint (passed from job executor)
func StreamToLogsChannel(resp io.Reader, logs chan output.Output) {

	scanner := bufio.NewScanner(resp)

	for scanner.Scan() {
		chunk := trimDataChunk(scanner.Bytes())

		// ignore lines which are not JSON objects
		if len(chunk) < 2 || chunk[0] != '{' {
			continue
		}

		// convert to output.Output object
		out := output.Output{}
		err := json.Unmarshal(chunk, &out)
		if err != nil {
			fmt.Printf("Unmarshal chunk error: %+v, json:'%s' \n", err, chunk)
			continue
		}

		logs <- out
	}
}

func trimDataChunk(in []byte) []byte {
	prefix := []byte("data: ")
	postfix := []byte("\\n\\n")
	chunk := bytes.Replace(in, prefix, []byte{}, 1)
	chunk = bytes.Replace(chunk, postfix, []byte{}, 1)

	// chunk = chunk[:len(chunk)-4]

	return chunk
}
