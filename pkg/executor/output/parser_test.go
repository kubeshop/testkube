package output

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

var exampleOutput = []byte(`
{"type":"event","message":"running postman/collection from testkube.Execution","content":["{\n\t\"id\": \"blablablablabla\",\n\t\"name\": \"some-testing-exec\",\n\t\"scriptContent\": \"{\\n\\t\\\"info\\\": {\\n\\t\\t\\\"_postman_id\\\": \\\"97b67bfb-c1ca-4572-af46-06cab8f68998\\\",\\n\\t\\t\\\"name\\\": \\\"Local-API-Health\\\",\\n\\t\\t\\\"schema\\\": \\\"https://schema.getpostman.com/json/collection/v2.1.0/collection.json\\\"\\n\\t},\\n\\t\\\"item\\\": [\\n\\t\\t{\\n\\t\\t\\t\\\"name\\\": \\\"Health\\\",\\n\\t\\t\\t\\\"event\\\": [\\n\\t\\t\\t\\t{\\n\\t\\t\\t\\t\\t\\\"listen\\\": \\\"test\\\",\\n\\t\\t\\t\\t\\t\\\"script\\\": {\\n\\t\\t\\t\\t\\t\\t\\\"exec\\\": [\\n\\t\\t\\t\\t\\t\\t\\t\\\"pm.test(\\\\\\\"Status code is 200\\\\\\\", function () {\\\",\\n\\t\\t\\t\\t\\t\\t\\t\\\"    pm.response.to.have.status(200);\\\",\\n\\t\\t\\t\\t\\t\\t\\t\\\"});\\\"\\n\\t\\t\\t\\t\\t\\t],\\n\\t\\t\\t\\t\\t\\t\\\"type\\\": \\\"text/javascript\\\"\\n\\t\\t\\t\\t\\t}\\n\\t\\t\\t\\t}\\n\\t\\t\\t],\\n\\t\\t\\t\\\"request\\\": {\\n\\t\\t\\t\\t\\\"method\\\": \\\"GET\\\",\\n\\t\\t\\t\\t\\\"header\\\": [],\\n\\t\\t\\t\\t\\\"url\\\": {\\n\\t\\t\\t\\t\\t\\\"raw\\\": \\\"http://localhost:8088/health\\\",\\n\\t\\t\\t\\t\\t\\\"protocol\\\": \\\"http\\\",\\n\\t\\t\\t\\t\\t\\\"host\\\": [\\n\\t\\t\\t\\t\\t\\t\\\"localhost\\\"\\n\\t\\t\\t\\t\\t],\\n\\t\\t\\t\\t\\t\\\"port\\\": \\\"8088\\\",\\n\\t\\t\\t\\t\\t\\\"path\\\": [\\n\\t\\t\\t\\t\\t\\t\\\"health\\\"\\n\\t\\t\\t\\t\\t]\\n\\t\\t\\t\\t}\\n\\t\\t\\t},\\n\\t\\t\\t\\\"response\\\": []\\n\\t\\t}\\n\\t],\\n\\t\\\"event\\\": []\\n}\"\n}"]}
{"type":"line","content":"newman\n\n"}
{"type":"line","content":"Local-API-Health\n"}
{"type":"line","content":"\n→ Health\n"}
{"type":"line","content":"  GET http://localhost:8088/health "}
{"type":"line","content":"[errored]\n"}
{"type":"line","content":"     connect ECONNREFUSED 127.0.0.1:8088\n"}
{"type":"line","content":"  2. Status code is 200\n"}
{"type":"line","content":"\n┌─────────────────────────┬──────────┬──────────┐\n│                         │ executed │   failed │\n├─────────────────────────┼──────────┼──────────┤\n│              iterations │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│                requests │        1 │        1 │\n├─────────────────────────┼──────────┼──────────┤\n│            test-scripts │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│      prerequest-scripts │        0 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│              assertions │        1 │        1 │\n├─────────────────────────┴──────────┴──────────┤\n│ total run duration: 1012ms                    │\n├───────────────────────────────────────────────┤\n│ total data received: 0B (approx)              │\n└───────────────────────────────────────────────┘\n"}
{"type":"line","content":"\n  #  failure         detail                                                          \n                                                                                     \n 1.  Error                                                                           \n                     connect ECONNREFUSED 127.0.0.1:8088                             \n                     at request                                                      \n                     inside \"Health\"                                                 \n                                                                                     \n 2.  AssertionError  Status code is 200                                              \n                     expected { Object (id, _details, ...) } to have property 'code' \n                     at assertion:0 in test-script                                   \n                     inside \"Health\"                                                 \n"}
{"type":"result","result":{"status":"failed","startTime":"2021-10-29T11:35:35.759Z","endTime":"2021-10-29T11:35:36.771Z","output":"newman\n\nLocal-API-Health\n\n→ Health\n  GET http://localhost:8088/health [errored]\n     connect ECONNREFUSED 127.0.0.1:8088\n  2. Status code is 200\n\n┌─────────────────────────┬──────────┬──────────┐\n│                         │ executed │   failed │\n├─────────────────────────┼──────────┼──────────┤\n│              iterations │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│                requests │        1 │        1 │\n├─────────────────────────┼──────────┼──────────┤\n│            test-scripts │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│      prerequest-scripts │        0 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│              assertions │        1 │        1 │\n├─────────────────────────┴──────────┴──────────┤\n│ total run duration: 1012ms                    │\n├───────────────────────────────────────────────┤\n│ total data received: 0B (approx)              │\n└───────────────────────────────────────────────┘\n\n  #  failure         detail                                                          \n                                                                                     \n 1.  Error                                                                           \n                     connect ECONNREFUSED 127.0.0.1:8088                             \n                     at request                                                      \n                     inside \"Health\"                                                 \n                                                                                     \n 2.  AssertionError  Status code is 200                                              \n                     expected { Object (id, _details, ...) } to have property 'code' \n                     at assertion:0 in test-script                                   \n                     inside \"Health\"                                                 \n","outputType":"text/plain","errorMessage":"process error: exit status 1","steps":[{"name":"Health","duration":"0s","status":"failed","assertionResults":[{"name":"Status code is 200","status":"failed","errorMessage":"expected { Object (id, _details, ...) } to have property 'code'"}]}]}}
`)

var exampleLogEntryLine = []byte(`{"type":"line","content":"  GET http://localhost:8088/health "}`)
var exampleLogEntryEvent = []byte(`{"type":"event","content":"running postman/collection from testkube.Execution","obj":["{\n\t\"id\": \"blablablablabla\",\n\t\"name\": \"some-testing-exec\",\n\t\"scriptContent\": \"{\\n\\t\\\"info\\\": {\\n\\t\\t\\\"_postman_id\\\": \\\"97b67bfb-c1ca-4572-af46-06cab8f68998\\\",\\n\\t\\t\\\"name\\\": \\\"Local-API-Health\\\",\\n\\t\\t\\\"schema\\\": \\\"https://schema.getpostman.com/json/collection/v2.1.0/collection.json\\\"\\n\\t},\\n\\t\\\"item\\\": [\\n\\t\\t{\\n\\t\\t\\t\\\"name\\\": \\\"Health\\\",\\n\\t\\t\\t\\\"event\\\": [\\n\\t\\t\\t\\t{\\n\\t\\t\\t\\t\\t\\\"listen\\\": \\\"test\\\",\\n\\t\\t\\t\\t\\t\\\"script\\\": {\\n\\t\\t\\t\\t\\t\\t\\\"exec\\\": [\\n\\t\\t\\t\\t\\t\\t\\t\\\"pm.test(\\\\\\\"Status code is 200\\\\\\\", function () {\\\",\\n\\t\\t\\t\\t\\t\\t\\t\\\"    pm.response.to.have.status(200);\\\",\\n\\t\\t\\t\\t\\t\\t\\t\\\"});\\\"\\n\\t\\t\\t\\t\\t\\t],\\n\\t\\t\\t\\t\\t\\t\\\"type\\\": \\\"text/javascript\\\"\\n\\t\\t\\t\\t\\t}\\n\\t\\t\\t\\t}\\n\\t\\t\\t],\\n\\t\\t\\t\\\"request\\\": {\\n\\t\\t\\t\\t\\\"method\\\": \\\"GET\\\",\\n\\t\\t\\t\\t\\\"header\\\": [],\\n\\t\\t\\t\\t\\\"url\\\": {\\n\\t\\t\\t\\t\\t\\\"raw\\\": \\\"http://localhost:8088/health\\\",\\n\\t\\t\\t\\t\\t\\\"protocol\\\": \\\"http\\\",\\n\\t\\t\\t\\t\\t\\\"host\\\": [\\n\\t\\t\\t\\t\\t\\t\\\"localhost\\\"\\n\\t\\t\\t\\t\\t],\\n\\t\\t\\t\\t\\t\\\"port\\\": \\\"8088\\\",\\n\\t\\t\\t\\t\\t\\\"path\\\": [\\n\\t\\t\\t\\t\\t\\t\\\"health\\\"\\n\\t\\t\\t\\t\\t]\\n\\t\\t\\t\\t}\\n\\t\\t\\t},\\n\\t\\t\\t\\\"response\\\": []\\n\\t\\t}\\n\\t],\\n\\t\\\"event\\\": []\\n}\"\n}"]}`)
var exampleLogEntryError = []byte(`{"type":"error","content":"Some error message"}`)
var exampleLogEntryResult = []byte(`{"type":"result","result":{"status":"failed","startTime":"2021-10-29T11:35:35.759Z","endTime":"2021-10-29T11:35:36.771Z","output":"newman\n\nLocal-API-Health\n\n→ Health\n  GET http://localhost:8088/health [errored]\n     connect ECONNREFUSED 127.0.0.1:8088\n  2. Status code is 200\n\n┌─────────────────────────┬──────────┬──────────┐\n│                         │ executed │   failed │\n├─────────────────────────┼──────────┼──────────┤\n│              iterations │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│                requests │        1 │        1 │\n├─────────────────────────┼──────────┼──────────┤\n│            test-scripts │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│      prerequest-scripts │        0 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│              assertions │        1 │        1 │\n├─────────────────────────┴──────────┴──────────┤\n│ total run duration: 1012ms                    │\n├───────────────────────────────────────────────┤\n│ total data received: 0B (approx)              │\n└───────────────────────────────────────────────┘\n\n  #  failure         detail                                                          \n                                                                                     \n 1.  Error                                                                           \n                     connect ECONNREFUSED 127.0.0.1:8088                             \n                     at request                                                      \n                     inside \"Health\"                                                 \n                                                                                     \n 2.  AssertionError  Status code is 200                                              \n                     expected { Object (id, _details, ...) } to have property 'code' \n                     at assertion:0 in test-script                                   \n                     inside \"Health\"                                                 \n","outputType":"text/plain","errorMessage":"process error: exit status 1","steps":[{"name":"Health","duration":"0s","status":"failed","assertionResults":[{"name":"Status code is 200","status":"failed","errorMessage":"expected { Object (id, _details, ...) } to have property 'code'"}]}]}}`)

func TestGetLogEntry(t *testing.T) {

	t.Run("get log line", func(t *testing.T) {
		out, err := GetLogEntry(exampleLogEntryLine)
		assert.NoError(t, err)
		assert.Equal(t, TypeLogLine, out.Type_)
	})

	t.Run("get event", func(t *testing.T) {
		out, err := GetLogEntry(exampleLogEntryEvent)
		assert.NoError(t, err)
		assert.Equal(t, TypeLogEvent, out.Type_)
	})

	t.Run("get error", func(t *testing.T) {
		out, err := GetLogEntry(exampleLogEntryError)
		assert.NoError(t, err)
		assert.Equal(t, TypeError, out.Type_)
	})

	t.Run("get result", func(t *testing.T) {
		out, err := GetLogEntry(exampleLogEntryResult)
		assert.NoError(t, err)
		assert.Equal(t, TypeResult, out.Type_)
	})

}

func TestParseRunnerOutput(t *testing.T) {

	result, logs, err := ParseRunnerOutput(exampleOutput)

	assert.Len(t, logs, 10)
	assert.NoError(t, err)
	assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
}
