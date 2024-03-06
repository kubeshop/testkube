package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/utils"
)

// Version is client version literal
const Version = "v1"

// TestkubeInstallationNamespace where Testkube is installed
const TestkubeInstallationNamespace = "testkube"

type TestingType string

const (
	Test      TestingType = "test"
	Execution TestingType = "execution"
)

// StreamToLogsChannel converts io.Reader with SSE data like `data: {"type": "event", "message":"something"}`
// to channel of output.Output objects, helps with logs streaming from SSE endpoint (passed from job executor)
func StreamToLogsChannel(resp io.Reader, logs chan output.Output) {
	reader := bufio.NewReader(resp)

	for {
		b, err := utils.ReadLongLine(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read long line error: %+v' \n", err)
			}

			break
		}
		chunk := trimDataChunk(b)

		// ignore lines which are not JSON objects
		if len(chunk) < 2 || chunk[0] != '{' {
			continue
		}

		// convert to output.Output object
		out := output.Output{}
		err = json.Unmarshal(chunk, &out)
		if err != nil {
			fmt.Printf("Unmarshal chunk error: %+v, json:'%s' \n", err, chunk)
			continue
		}

		logs <- out
	}
}

// StreamToLogsChannelV2 converts io.Reader with SSE data like `data: {"type": "event", "message":"something"}`
// to channel of output.Output objects, helps with logs version 2 streaming from SSE endpoint (passed from job executor)
func StreamToLogsChannelV2(resp io.Reader, logs chan events.Log) {
	reader := bufio.NewReader(resp)

	for {
		b, err := utils.ReadLongLine(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read long line error: %+v' \n", err)
			}

			break
		}
		chunk := trimDataChunk(b)

		// ignore lines which are not JSON objects
		if len(chunk) < 2 || chunk[0] != '{' {
			continue
		}

		// convert to events.Log object
		out := events.Log{}
		err = json.Unmarshal(chunk, &out)
		if err != nil {
			fmt.Printf("Unmarshal chunk error: %+v, json:'%s' \n", err, chunk)
			continue
		}

		logs <- out
	}
}

// StreamToTestWorkflowExecutionNotificationsChannel converts io.Reader with SSE data to channel of actual notifications
func StreamToTestWorkflowExecutionNotificationsChannel(resp io.Reader, notifications chan testkube.TestWorkflowExecutionNotification) {
	reader := bufio.NewReader(resp)

	for {
		b, err := utils.ReadLongLine(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read long line error: %+v' \n", err)
			}

			break
		}
		chunk := trimDataChunk(b)

		// ignore lines which are not JSON objects
		if len(chunk) < 2 || chunk[0] != '{' {
			continue
		}

		out := testkube.TestWorkflowExecutionNotification{}
		err = json.Unmarshal(chunk, &out)
		if err != nil {
			fmt.Printf("Unmarshal chunk error: %+v, json:'%s' \n", err, chunk)
			continue
		}

		notifications <- out
	}
}

// trimDataChunk remove data: and newlines from incoming SSE data line
func trimDataChunk(in []byte) []byte {
	prefix := []byte("data: ")
	postfix := []byte("\\n\\n")
	chunk := bytes.Replace(in, prefix, []byte{}, 1)
	chunk = bytes.Replace(chunk, postfix, []byte{}, 1)

	return chunk
}
