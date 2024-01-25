package output

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var exampleLogEntryLine = []byte(`{"type":"line","content":"  GET http://localhost:8088/health "}`)
var exampleLogEntryEvent = []byte(`{"type":"event","content":"running postman/collection from testkube.Execution","obj":["{\n\t\"id\": \"blablablablabla\",\n\t\"name\": \"some-testing-exec\",\n\t\"scriptContent\": \"{\\n\\t\\\"info\\\": {\\n\\t\\t\\\"_postman_id\\\": \\\"97b67bfb-c1ca-4572-af46-06cab8f68998\\\",\\n\\t\\t\\\"name\\\": \\\"Local-API-Health\\\",\\n\\t\\t\\\"schema\\\": \\\"https://schema.getpostman.com/json/collection/v2.1.0/collection.json\\\"\\n\\t},\\n\\t\\\"item\\\": [\\n\\t\\t{\\n\\t\\t\\t\\\"name\\\": \\\"Health\\\",\\n\\t\\t\\t\\\"event\\\": [\\n\\t\\t\\t\\t{\\n\\t\\t\\t\\t\\t\\\"listen\\\": \\\"test\\\",\\n\\t\\t\\t\\t\\t\\\"script\\\": {\\n\\t\\t\\t\\t\\t\\t\\\"exec\\\": [\\n\\t\\t\\t\\t\\t\\t\\t\\\"pm.test(\\\\\\\"Status code is 200\\\\\\\", function () {\\\",\\n\\t\\t\\t\\t\\t\\t\\t\\\"    pm.response.to.have.status(200);\\\",\\n\\t\\t\\t\\t\\t\\t\\t\\\"});\\\"\\n\\t\\t\\t\\t\\t\\t],\\n\\t\\t\\t\\t\\t\\t\\\"type\\\": \\\"text/javascript\\\"\\n\\t\\t\\t\\t\\t}\\n\\t\\t\\t\\t}\\n\\t\\t\\t],\\n\\t\\t\\t\\\"request\\\": {\\n\\t\\t\\t\\t\\\"method\\\": \\\"GET\\\",\\n\\t\\t\\t\\t\\\"header\\\": [],\\n\\t\\t\\t\\t\\\"url\\\": {\\n\\t\\t\\t\\t\\t\\\"raw\\\": \\\"http://localhost:8088/health\\\",\\n\\t\\t\\t\\t\\t\\\"protocol\\\": \\\"http\\\",\\n\\t\\t\\t\\t\\t\\\"host\\\": [\\n\\t\\t\\t\\t\\t\\t\\\"localhost\\\"\\n\\t\\t\\t\\t\\t],\\n\\t\\t\\t\\t\\t\\\"port\\\": \\\"8088\\\",\\n\\t\\t\\t\\t\\t\\\"path\\\": [\\n\\t\\t\\t\\t\\t\\t\\\"health\\\"\\n\\t\\t\\t\\t\\t]\\n\\t\\t\\t\\t}\\n\\t\\t\\t},\\n\\t\\t\\t\\\"response\\\": []\\n\\t\\t}\\n\\t],\\n\\t\\\"event\\\": []\\n}\"\n}"]}`)
var exampleLogEntryError = []byte(`{"type":"error","content":"Some error message"}`)
var exampleLogEntryResult = []byte(`{"type":"result","result":{"status":"failed","startTime":"2021-10-29T11:35:35.759Z","endTime":"2021-10-29T11:35:36.771Z","output":"newman\n\nLocal-API-Health\n\n→ Health\n  GET http://localhost:8088/health [errored]\n     connect ECONNREFUSED 127.0.0.1:8088\n  2. Status code is 200\n\n┌─────────────────────────┬──────────┬──────────┐\n│                         │ executed │   failed │\n├─────────────────────────┼──────────┼──────────┤\n│              iterations │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│                requests │        1 │        1 │\n├─────────────────────────┼──────────┼──────────┤\n│            test-scripts │        1 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│      prerequest-scripts │        0 │        0 │\n├─────────────────────────┼──────────┼──────────┤\n│              assertions │        1 │        1 │\n├─────────────────────────┴──────────┴──────────┤\n│ total run duration: 1012ms                    │\n├───────────────────────────────────────────────┤\n│ total data received: 0B (approx)              │\n└───────────────────────────────────────────────┘\n\n  #  failure         detail                                                          \n                                                                                     \n 1.  Error                                                                           \n                     connect ECONNREFUSED 127.0.0.1:8088                             \n                     at request                                                      \n                     inside \"Health\"                                                 \n                                                                                     \n 2.  AssertionError  Status code is 200                                              \n                     expected { Object (id, _details, ...) } to have property 'code' \n                     at assertion:0 in test-script                                   \n                     inside \"Health\"                                                 \n","outputType":"text/plain","errorMessage":"process error: exit status 1","steps":[{"name":"Health","duration":"0s","status":"failed","assertionResults":[{"name":"Status code is 200","status":"failed","errorMessage":"expected { Object (id, _details, ...) } to have property 'code'"}]}]}}`)
var exampleLogEntryNormalLogs = []byte(`{"level":"info","ts":1680606449.51631,"caller":"scraper/filesystem_extractor.go:37","msg":"walking path /data/output/report/sbadmin2-1.0.7/dist"}`)

func TestGetLogEntry(t *testing.T) {
	t.Parallel()

	t.Run("get log line", func(t *testing.T) {
		t.Parallel()

		out, err := GetLogEntry(exampleLogEntryLine)
		assert.NoError(t, err)
		assert.Equal(t, TypeLogLine, out.Type_)
	})

	t.Run("get event", func(t *testing.T) {
		t.Parallel()

		out, err := GetLogEntry(exampleLogEntryEvent)
		assert.NoError(t, err)
		assert.Equal(t, TypeLogEvent, out.Type_)
	})

	t.Run("get error", func(t *testing.T) {
		t.Parallel()

		out, err := GetLogEntry(exampleLogEntryError)
		assert.NoError(t, err)
		assert.Equal(t, TypeError, out.Type_)
	})

	t.Run("get result", func(t *testing.T) {
		t.Parallel()

		out, err := GetLogEntry(exampleLogEntryResult)
		assert.NoError(t, err)
		assert.Equal(t, TypeResult, out.Type_)
	})

	t.Run("get normal logs", func(t *testing.T) {
		t.Parallel()

		out, err := GetLogEntry(exampleLogEntryNormalLogs)
		assert.NoError(t, err)
		assert.Equal(t, TypeUnknown, out.Type_)
	})
}

func TestParseRunnerOutput(t *testing.T) {
	t.Parallel()

	t.Run("Empty runner output", func(t *testing.T) {
		t.Parallel()

		result, err := ParseRunnerOutput([]byte{}, true)

		assert.Equal(t, "no logs found", result.Output)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
	})

	t.Run("Invalid log", func(t *testing.T) {
		t.Parallel()

		invalidOutput := []byte(`{not a json}`)
		result, err := ParseRunnerOutput(invalidOutput, true)
		expectedErrMessage := "ERROR can't get log entry: invalid character 'n' looking for beginning of object key string, ((({not a json})))"

		assert.Equal(t, expectedErrMessage+"\n", result.Output)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, expectedErrMessage, result.ErrorMessage)
	})

	t.Run("Runner output with no timestamps", func(t *testing.T) {
		t.Parallel()

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
		result, err := ParseRunnerOutput(exampleOutput, true)

		assert.Len(t, result.Output, 4624)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "process error: exit status 1", result.ErrorMessage)
	})

	t.Run("Runner output with passed status", func(t *testing.T) {
		t.Parallel()

		var exampleOutput = []byte(`
{"type":"line","content":"🌍 Reading environment variables...","time":"2023-07-18T19:12:44.065916596Z"} 
{"type":"line","content":"✅ Environment variables read successfully","time":"2023-07-18T19:12:44.066174662Z"} 
{"type":"line","content":"RUNNER_ENDPOINT=\"testkube-minio-service-shared:9000\"","time":"2023-07-18T19:12:44.066185893Z"} 
{"type":"line","content":"RUNNER_ACCESSKEYID=\"********\"","time":"2023-07-18T19:12:44.066190525Z"} 
{"type":"line","content":"RUNNER_SECRETACCESSKEY=\"********\"","time":"2023-07-18T19:12:44.066198451Z"} 
{"type":"line","content":"RUNNER_REGION=\"\"","time":"2023-07-18T19:12:44.066202295Z"} 
{"type":"line","content":"RUNNER_TOKEN=\"\"","time":"2023-07-18T19:12:44.066206054Z"} 
{"type":"line","content":"RUNNER_BUCKET=\"testkube-artifacts\"","time":"2023-07-18T19:12:44.066209522Z"} 
{"type":"line","content":"RUNNER_SSL=false","time":"2023-07-18T19:12:44.066215367Z"} 
{"type":"line","content":"RUNNER_SCRAPPERENABLED=\"true\"","time":"2023-07-18T19:12:44.066218487Z"} 
{"type":"line","content":"RUNNER_GITUSERNAME=\"********\"","time":"2023-07-18T19:12:44.066222184Z"} 
{"type":"line","content":"RUNNER_GITTOKEN=\"********\"","time":"2023-07-18T19:12:44.066226231Z"} 
{"type":"line","content":"RUNNER_DATADIR=\"/data\"","time":"2023-07-18T19:12:44.066229687Z"} 
{"type":"line","content":"RUNNER_CLUSTERID=\"clusterf83c3172f255fad68c2ab1a59be52916\"","time":"2023-07-18T19:12:44.066233268Z"} 
{"type":"line","content":"RUNNER_CDEVENTS_TARGET=\"\"","time":"2023-07-18T19:12:44.066237063Z"} 
{"type":"line","content":"RUNNER_DASHBOARD_URI=\"\"","time":"2023-07-18T19:12:44.066240323Z"} 
{"type":"line","content":"RUNNER_CLOUD_MODE=\"false\"","time":"2023-07-18T19:12:44.066244115Z"} 
{"type":"line","content":"RUNNER_CLOUD_API_TLS_INSECURE=\"false\"","time":"2023-07-18T19:12:44.066247795Z"} 
{"type":"line","content":"RUNNER_CLOUD_API_URL=\"agent.testkube.io:443\"","time":"2023-07-18T19:12:44.066251778Z"} 
{"type":"line","content":"RUNNER_CLOUD_API_KEY=\"\"","time":"2023-07-18T19:12:44.066255395Z"} 
{"type":"line","content":"RUNNER_CLOUD_CONNECTION_TIMEOUT=10","time":"2023-07-18T19:12:44.066259384Z"} 
{"type":"line","content":"🚚 Preparing test runner","time":"2023-07-18T19:12:44.066263179Z"} 
{"type":"line","content":"✅ Uploading artifacts using MinIO Uploader","time":"2023-07-18T19:12:44.066269065Z"} 
{"type":"event","content":"running test [64b6e3f0530875ebaf73794f]","time":"2023-07-18T19:12:44.06658648Z"} 
{"type":"line","content":"🚚 Preparing for test run","time":"2023-07-18T19:12:44.066599252Z"} 
{"type":"line","content":"🚀 Test run command newman run /data/repo/Core App Tests - WebPlayer.postman_collection.json -e /tmp/testkube-tmp1897306484 --reporters cli,json --reporter-json-export /tmp/testkube-tmp4151526495.json","time":"2023-07-18T19:12:44.066856036Z"} 
{"type":"line","content":"🔬 Executing in directory : \n $ newman run /data/repo/Core App Tests - WebPlayer.postman_collection.json -e /tmp/testkube-tmp1897306484 --reporters cli,json --reporter-json-export /tmp/testkube-tmp4151526495.json","time":"2023-07-18T19:12:44.066872166Z"} 
{"type":"line","content":"newman\n\n","time":"2023-07-18T19:12:45.654031992Z"} 
{"type":"line","content":"Core App Tests - WebPlayer\n","time":"2023-07-18T19:12:45.654989975Z"} 
{"type":"line","content":"\n→ na.com client=testdb sign=testct1 company=41574150-b952-413b-898b-dc5336b4bd12\n","time":"2023-07-18T19:12:45.658607412Z"} 
{"type":"line","content":"  GET https://na.com/v6-wplt/?client=testdb\u0026sign=testct1\u0026company=41574150-b952-413b-898b-dc5336b4bd12 ","time":"2023-07-18T19:12:45.685011667Z"} 
{"type":"line","content":"[200 OK, 33.9kB, 326ms]\n","time":"2023-07-18T19:12:46.014247794Z"} 
{"type":"line","content":"  ✓  Status code is 200\n","time":"2023-07-18T19:12:46.076176552Z"}
{"type":"line","content":"\n┌─────────────────────────┬────────────────────┬───────────────────┐\n│                         │           executed │            failed │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              iterations │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│                requests │                  1 │   0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│            test-scripts │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│      prerequest-scripts │                  0 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              assertions │                  1 │ 0 │\n├─────────────────────────┴────────────────────┴───────────────────┤\n│ total run duration: 429ms                                        │\n├──────────────────────────────────────────────────────────────────┤\n│ total data received: 33.45kB (approx)                            │\n├──────────────────────────────────────────────────────────────────┤\n│ average response time: 326ms [min: 326ms, max: 326ms, s.d.: 0µs] │\n└──────────────────────────────────────────────────────────────────┘\n","time":"2023-07-18T19:12:46.104174132Z"} 
{"type":"line","content":"✅ Execution succeeded","time":"2023-07-18T19:12:46.114565626Z"} 
{"type":"line","content":"✅ Got Newman result successfully","time":"2023-07-18T19:12:46.126116248Z"} 
{"type":"line","content":"✅ Mapped Newman result successfully","time":"2023-07-18T19:12:46.126152021Z"} 
{"type":"result","result":{"status":"passed","output":"newman\n\nCore App Tests - WebPlayer\n\n→ core-eks-test.poppcore.co client=testdb sign=testct1 company=41574150-b952-413b-898b-dc5336b4bd12\n  GET https://na.com/v6-wplt/?client=testdb\u0026sign=testct1\u0026company=41574150-b952-413b-898b-dc5336b4bd12 [200 OK, 33.9kB, 326ms]\n  ✓  Status code is 200\n\n┌─────────────────────────┬────────────────────┬───────────────────┐\n│                         │           executed │            failed │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              iterations │                  1 │      0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│                requests │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│            test-scripts │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│      prerequest-scripts │                  0 │ 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              assertions │                  1 │                 0 │\n├─────────────────────────┴────────────────────┴───────────────────┤\n│ total run duration: 429ms                                        │\n├──────────────────────────────────────────────────────────────────┤\n│ total data received: 33.45kB (approx)                            │\n├──────────────────────────────────────────────────────────────────┤\n│ average response time: 326ms [min: 326ms, max: 326ms, s.d.: 0µs] │\n└──────────────────────────────────────────────────────────────────┘\n","outputType":"text/plain","steps":[{"name":"na.com client=testdb sign=testct1 company=41574150-b952-413b-898b-dc5336b4bd12","duration":"326ms","status":"passed","assertionResults":[{"name":"Status code is 200","status":"passed"}]}]},"time":"2023-07-18T19:12:46.12615853Z"}`)
		result, err := ParseRunnerOutput(exampleOutput, true)

		assert.Len(t, result.Output, 7304)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
	})

	t.Run("Wrong order in logs", func(t *testing.T) {
		t.Parallel()

		unorderedOutput := []byte(`
{"type":"line","content":"🚚 Preparing test runner","time":"2023-01-17T15:29:17.921466388Z"}
{"type":"line","content":"🌍 Reading environment variables...","time":"2023-01-17T15:29:17.921770638Z"}
{"type":"line","content":"✅ Environment variables read successfully","time":"2023-01-17T15:29:17.921786721Z"}
{"type":"line","content":"RUNNER_ENDPOINT=\"testkube-minio-service-testkube:9000\"","time":"2023-01-17T15:29:17.921788721Z"}
{"type":"line","content":"RUNNER_ACCESSKEYID=\"********\"","time":"2023-01-17T15:29:17.921790721Z"}
{"type":"line","content":"RUNNER_SECRETACCESSKEY=\"********\"","time":"2023-01-17T15:29:17.921792388Z"}
{"type":"line","content":"RUNNER_REGION=\"\"","time":"2023-01-17T15:29:17.921793846Z"}
{"type":"line","content":"RUNNER_TOKEN=\"\"","time":"2023-01-17T15:29:17.921795304Z"}
{"type":"line","content":"RUNNER_SSL=false","time":"2023-01-17T15:29:17.921797054Z"}
{"type":"line","content":"RUNNER_SCRAPPERENABLED=\"true\"","time":"2023-01-17T15:29:17.921798679Z"}
{"type":"line","content":"RUNNER_GITUSERNAME=\"\"","time":"2023-01-17T15:29:17.921800138Z"}
{"type":"line","content":"RUNNER_GITTOKEN=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_DATADIR=\"/data\"","time":"2023-01-17T15:29:17.921803138Z"}
{"type":"line","content":"RUNNER_CDEVENTS_TARGET=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_DASHBOARD_URI=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_CLUSTERID=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"error","content":"❌ can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-17T15:29:17.921940304Z"}
{"type":"error","content":"can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-17T15:29:17.921946638Z"}
{"type":"event","content":"running test [63c6bec1790802b7e3e57048]","time":"2023-01-17T15:29:17.921920596Z"}
{"type":"line","content":"🚚 Preparing for test run","time":"2023-01-17T15:29:17.921931471Z"}
`)
		expectedOutput := `🚚 Preparing test runner
🌍 Reading environment variables...
✅ Environment variables read successfully
RUNNER_ENDPOINT="testkube-minio-service-testkube:9000"
RUNNER_ACCESSKEYID="********"
RUNNER_SECRETACCESSKEY="********"
RUNNER_REGION=""
RUNNER_TOKEN=""
RUNNER_SSL=false
RUNNER_SCRAPPERENABLED="true"
RUNNER_GITUSERNAME=""
RUNNER_GITTOKEN=""
RUNNER_DATADIR="/data"
RUNNER_CDEVENTS_TARGET=""
RUNNER_DASHBOARD_URI=""
RUNNER_CLUSTERID=""
❌ can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
running test [63c6bec1790802b7e3e57048]
🚚 Preparing for test run
`
		result, err := ParseRunnerOutput(unorderedOutput, true)

		assert.Equal(t, expectedOutput, result.Output)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}", result.ErrorMessage)
	})

	t.Run("Output with failed result", func(t *testing.T) {
		t.Parallel()

		output := []byte(`{"type":"event","content":"running test [63c9606d7104b0fa0b7a45f1]"}
{"type":"event","content":"Running [ ./zap-api-scan.py [-t https://www.example.com/openapi.json -f openapi -c examples/zap-api.conf -d -D 5 -I -l INFO -n examples/context.config -S -T 60 -U anonymous -O https://www.example.com -z -config aaa=bbb -r api-test-report.html]]"}
{"type":"result","result":{"status":"failed","errorMessage":"could not start process: fork/exec ./zap-api-scan.py: no such file or directory"}}`)
		expectedOutput := `running test [63c9606d7104b0fa0b7a45f1]
Running [ ./zap-api-scan.py [-t https://www.example.com/openapi.json -f openapi -c examples/zap-api.conf -d -D 5 -I -l INFO -n examples/context.config -S -T 60 -U anonymous -O https://www.example.com -z -config aaa=bbb -r api-test-report.html]]
could not start process: fork/exec ./zap-api-scan.py: no such file or directory
`
		result, err := ParseRunnerOutput(output, true)

		assert.Equal(t, expectedOutput, result.Output)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "could not start process: fork/exec ./zap-api-scan.py: no such file or directory", result.ErrorMessage)
	})

	t.Run("Output with error", func(t *testing.T) {
		t.Parallel()

		output := []byte(`
{"type":"line","content":"🚚 Preparing test runner","time":"2023-01-19T15:22:25.867379429Z"}
{"type":"line","content":"🌍 Reading environment variables...","time":"2023-01-19T15:22:25.867927513Z"}
{"type":"line","content":"✅ Environment variables read successfully","time":"2023-01-19T15:22:25.867944763Z"}
{"type":"line","content":"RUNNER_ENDPOINT=\"testkube-minio-service-testkube:9000\"","time":"2023-01-19T15:22:25.867946929Z"}
{"type":"line","content":"RUNNER_ACCESSKEYID=\"********\"","time":"2023-01-19T15:22:25.867948804Z"}
{"type":"line","content":"RUNNER_SECRETACCESSKEY=\"********\"","time":"2023-01-19T15:22:25.867955263Z"}
{"type":"line","content":"RUNNER_REGION=\"\"","time":"2023-01-19T15:22:25.867962596Z"}
{"type":"line","content":"RUNNER_TOKEN=\"\"","time":"2023-01-19T15:22:25.867967971Z"}
{"type":"line","content":"RUNNER_SSL=false","time":"2023-01-19T15:22:25.867974013Z"}
{"type":"line","content":"RUNNER_SCRAPPERENABLED=\"true\"","time":"2023-01-19T15:22:25.867978888Z"}
{"type":"line","content":"RUNNER_GITUSERNAME=\"\"","time":"2023-01-19T15:22:25.867984179Z"}
{"type":"line","content":"RUNNER_GITTOKEN=\"\"","time":"2023-01-19T15:22:25.867986013Z"}
{"type":"line","content":"RUNNER_DATADIR=\"/data\"","time":"2023-01-19T15:22:25.867987596Z"}
{"type":"line","content":"RUNNER_CDEVENTS_TARGET=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_DASHBOARD_URI=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_CLUSTERID=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"event","content":"running test [63c960287104b0fa0b7a45ef]","time":"2023-01-19T15:22:25.868132888Z"}
{"type":"line","content":"🚚 Preparing for test run","time":"2023-01-19T15:22:25.868161346Z"}
{"type":"line","content":"❌ can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-19T15:22:25.868183971Z"}
{"type":"error","content":"can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-19T15:22:25.868198429Z"}
`)
		expectedOutput := `🚚 Preparing test runner
🌍 Reading environment variables...
✅ Environment variables read successfully
RUNNER_ENDPOINT="testkube-minio-service-testkube:9000"
RUNNER_ACCESSKEYID="********"
RUNNER_SECRETACCESSKEY="********"
RUNNER_REGION=""
RUNNER_TOKEN=""
RUNNER_SSL=false
RUNNER_SCRAPPERENABLED="true"
RUNNER_GITUSERNAME=""
RUNNER_GITTOKEN=""
RUNNER_DATADIR="/data"
RUNNER_CDEVENTS_TARGET=""
RUNNER_DASHBOARD_URI=""
RUNNER_CLUSTERID=""
running test [63c960287104b0fa0b7a45ef]
🚚 Preparing for test run
❌ can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
`

		result, err := ParseRunnerOutput(output, true)

		assert.Equal(t, expectedOutput, result.Output)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}", result.ErrorMessage)
	})

	t.Run("Remove internal logs", func(t *testing.T) {
		t.Parallel()

		output := []byte(`
{"type":"line","content":"🚚 Preparing test runner","time":"2023-01-19T15:22:25.867379429Z"}
{"type":"line","content":"🌍 Reading environment variables...","time":"2023-01-19T15:22:25.867927513Z"}
{"type":"line","content":"✅ Environment variables read successfully","time":"2023-01-19T15:22:25.867944763Z"}
{"type":"line","content":"RUNNER_ENDPOINT=\"testkube-minio-service-testkube:9000\"","time":"2023-01-19T15:22:25.867946929Z"}
{"type":"line","content":"RUNNER_ACCESSKEYID=\"********\"","time":"2023-01-19T15:22:25.867948804Z"}
{"type":"line","content":"RUNNER_SECRETACCESSKEY=\"********\"","time":"2023-01-19T15:22:25.867955263Z"}
{"type":"line","content":"RUNNER_REGION=\"\"","time":"2023-01-19T15:22:25.867962596Z"}
{"type":"line","content":"RUNNER_TOKEN=\"\"","time":"2023-01-19T15:22:25.867967971Z"}
{"type":"line","content":"RUNNER_SSL=false","time":"2023-01-19T15:22:25.867974013Z"}
{"type":"line","content":"RUNNER_SCRAPPERENABLED=\"true\"","time":"2023-01-19T15:22:25.867978888Z"}
{"type":"line","content":"RUNNER_GITUSERNAME=\"\"","time":"2023-01-19T15:22:25.867984179Z"}
{"type":"line","content":"RUNNER_GITTOKEN=\"\"","time":"2023-01-19T15:22:25.867986013Z"}
{"type":"line","content":"RUNNER_DATADIR=\"/data\"","time":"2023-01-19T15:22:25.867987596Z"}
{"type":"line","content":"RUNNER_CDEVENTS_TARGET=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_DASHBOARD_URI=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_CLUSTERID=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"event","content":"running test [63c960287104b0fa0b7a45ef]","time":"2023-01-19T15:22:25.868132888Z"}
{"type":"line","content":"🚚 Preparing for test run","time":"2023-01-19T15:22:25.868161346Z"}
{"type":"line","content":"❌ can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-19T15:22:25.868183971Z"}
{"level":"info","ts":1678919600.1199453,"caller":"scraper/filesystem_extractor.go:25","msg":"extracting files from directories: [/data/output]"}
{"level":"info","ts":1678919600.143759,"caller":"scraper/filesystem_extractor.go:27","msg":"walking directory: /data/output"}
{"level":"info","ts":1678919600.145652,"caller":"scraper/filesystem_extractor.go:37","msg":"walking path /data/output"}
{"level":"info","ts":1678919600.1457458,"caller":"scraper/filesystem_extractor.go:43","msg":"skipping directory /data/output"}
{"level":"info","ts":1678919600.1459277,"caller":"scraper/filesystem_extractor.go:37","msg":"walking path /data/output/jmeter.log"}
{"level":"info","ts":1678919600.1480958,"caller":"scraper/filesystem_extractor.go:63","msg":"filesystem extractor is sending file to be processed: jmeter.log"}
{"type":"error","content":"can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-19T15:22:25.868198429Z"}
`)
		expectedOutput := `🚚 Preparing test runner
🌍 Reading environment variables...
✅ Environment variables read successfully
RUNNER_ENDPOINT="testkube-minio-service-testkube:9000"
RUNNER_ACCESSKEYID="********"
RUNNER_SECRETACCESSKEY="********"
RUNNER_REGION=""
RUNNER_TOKEN=""
RUNNER_SSL=false
RUNNER_SCRAPPERENABLED="true"
RUNNER_GITUSERNAME=""
RUNNER_GITTOKEN=""
RUNNER_DATADIR="/data"
RUNNER_CDEVENTS_TARGET=""
RUNNER_DASHBOARD_URI=""
RUNNER_CLUSTERID=""
running test [63c960287104b0fa0b7a45ef]
🚚 Preparing for test run
❌ can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
`

		result, err := ParseRunnerOutput(output, true)

		assert.Equal(t, expectedOutput, result.Output)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}", result.ErrorMessage)
	})

	t.Run("flaky cypress test", func(t *testing.T) {
		t.Parallel()

		output := []byte(`
{"type":"error","content":"can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-20T12:44:15.719459174Z"}
{"type":"line","content":"🚚 Preparing test runner","time":"2023-01-20T12:44:15.714299549Z"}
{"type":"line","content":"🌍 Reading environment variables...","time":"2023-01-20T12:44:15.718910883Z"}
{"type":"line","content":"✅ Environment variables read successfully","time":"2023-01-20T12:44:15.718958008Z"}
{"type":"line","content":"RUNNER_ENDPOINT=\"testkube-minio-service-testkube:9000\"","time":"2023-01-20T12:44:15.718962633Z"}
{"type":"line","content":"RUNNER_ACCESSKEYID=\"********\"","time":"2023-01-20T12:44:15.718966091Z"}
{"type":"line","content":"RUNNER_SECRETACCESSKEY=\"********\"","time":"2023-01-20T12:44:15.718969383Z"}
{"type":"line","content":"RUNNER_REGION=\"\"","time":"2023-01-20T12:44:15.718972299Z"}
{"type":"line","content":"RUNNER_TOKEN=\"\"","time":"2023-01-20T12:44:15.718975174Z"}
{"type":"line","content":"RUNNER_SSL=false","time":"2023-01-20T12:44:15.718977924Z"}
{"type":"line","content":"RUNNER_SCRAPPERENABLED=\"true\"","time":"2023-01-20T12:44:15.718980758Z"}
{"type":"line","content":"RUNNER_GITUSERNAME=\"\"","time":"2023-01-20T12:44:15.718983549Z"}
{"type":"line","content":"RUNNER_GITTOKEN=\"\"","time":"2023-01-20T12:44:15.718986174Z"}
{"type":"line","content":"RUNNER_DATADIR=\"/data\"","time":"2023-01-20T12:44:15.718989049Z"}
{"type":"line","content":"RUNNER_CDEVENTS_TARGET=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_DASHBOARD_URI=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_CLUSTERID=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"event","content":"running test [63ca8c8988564860327a16b5]","time":"2023-01-20T12:44:15.719276383Z"}
{"type":"line","content":"🚚 Preparing for test run","time":"2023-01-20T12:44:15.719285633Z"}
{"type":"line","content":"❌ can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-20T12:44:15.719302049Z"}
`)
		expectedOutput := `can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
🚚 Preparing test runner
🌍 Reading environment variables...
✅ Environment variables read successfully
RUNNER_ENDPOINT="testkube-minio-service-testkube:9000"
RUNNER_ACCESSKEYID="********"
RUNNER_SECRETACCESSKEY="********"
RUNNER_REGION=""
RUNNER_TOKEN=""
RUNNER_SSL=false
RUNNER_SCRAPPERENABLED="true"
RUNNER_GITUSERNAME=""
RUNNER_GITTOKEN=""
RUNNER_DATADIR="/data"
RUNNER_CDEVENTS_TARGET=""
RUNNER_DASHBOARD_URI=""
RUNNER_CLUSTERID=""
running test [63ca8c8988564860327a16b5]
🚚 Preparing for test run
❌ can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
`

		result, err := ParseRunnerOutput(output, true)

		assert.Equal(t, expectedOutput, result.Output)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}", result.ErrorMessage)

	})
}

func TestParseContainerOutput(t *testing.T) {
	t.Parallel()

	t.Run("Empty runner output", func(t *testing.T) {
		t.Parallel()

		result, output, err := ParseContainerOutput([]byte{})

		assert.Equal(t, "", output)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Runner output with no timestamps", func(t *testing.T) {
		t.Parallel()

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
		result, output, err := ParseContainerOutput(exampleOutput)

		assert.Len(t, output, 4511)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "process error: exit status 1", result.ErrorMessage)
	})

	t.Run("Runner output with passed status", func(t *testing.T) {
		t.Parallel()

		var exampleOutput = []byte(`
{"type":"line","content":"🌍 Reading environment variables...","time":"2023-07-18T19:12:44.065916596Z"} 
{"type":"line","content":"✅ Environment variables read successfully","time":"2023-07-18T19:12:44.066174662Z"} 
{"type":"line","content":"RUNNER_ENDPOINT=\"testkube-minio-service-shared:9000\"","time":"2023-07-18T19:12:44.066185893Z"} 
{"type":"line","content":"RUNNER_ACCESSKEYID=\"********\"","time":"2023-07-18T19:12:44.066190525Z"} 
{"type":"line","content":"RUNNER_SECRETACCESSKEY=\"********\"","time":"2023-07-18T19:12:44.066198451Z"} 
{"type":"line","content":"RUNNER_REGION=\"\"","time":"2023-07-18T19:12:44.066202295Z"} 
{"type":"line","content":"RUNNER_TOKEN=\"\"","time":"2023-07-18T19:12:44.066206054Z"} 
{"type":"line","content":"RUNNER_BUCKET=\"testkube-artifacts\"","time":"2023-07-18T19:12:44.066209522Z"} 
{"type":"line","content":"RUNNER_SSL=false","time":"2023-07-18T19:12:44.066215367Z"} 
{"type":"line","content":"RUNNER_SCRAPPERENABLED=\"true\"","time":"2023-07-18T19:12:44.066218487Z"} 
{"type":"line","content":"RUNNER_GITUSERNAME=\"********\"","time":"2023-07-18T19:12:44.066222184Z"} 
{"type":"line","content":"RUNNER_GITTOKEN=\"********\"","time":"2023-07-18T19:12:44.066226231Z"} 
{"type":"line","content":"RUNNER_DATADIR=\"/data\"","time":"2023-07-18T19:12:44.066229687Z"} 
{"type":"line","content":"RUNNER_CLUSTERID=\"clusterf83c3172f255fad68c2ab1a59be52916\"","time":"2023-07-18T19:12:44.066233268Z"} 
{"type":"line","content":"RUNNER_CDEVENTS_TARGET=\"\"","time":"2023-07-18T19:12:44.066237063Z"} 
{"type":"line","content":"RUNNER_DASHBOARD_URI=\"\"","time":"2023-07-18T19:12:44.066240323Z"} 
{"type":"line","content":"RUNNER_CLOUD_MODE=\"false\"","time":"2023-07-18T19:12:44.066244115Z"} 
{"type":"line","content":"RUNNER_CLOUD_API_TLS_INSECURE=\"false\"","time":"2023-07-18T19:12:44.066247795Z"} 
{"type":"line","content":"RUNNER_CLOUD_API_URL=\"agent.testkube.io:443\"","time":"2023-07-18T19:12:44.066251778Z"} 
{"type":"line","content":"RUNNER_CLOUD_API_KEY=\"\"","time":"2023-07-18T19:12:44.066255395Z"} 
{"type":"line","content":"RUNNER_CLOUD_CONNECTION_TIMEOUT=10","time":"2023-07-18T19:12:44.066259384Z"} 
{"type":"line","content":"🚚 Preparing test runner","time":"2023-07-18T19:12:44.066263179Z"} 
{"type":"line","content":"✅ Uploading artifacts using MinIO Uploader","time":"2023-07-18T19:12:44.066269065Z"} 
{"type":"event","content":"running test [64b6e3f0530875ebaf73794f]","time":"2023-07-18T19:12:44.06658648Z"} 
{"type":"line","content":"🚚 Preparing for test run","time":"2023-07-18T19:12:44.066599252Z"} 
{"type":"line","content":"🚀 Test run command newman run /data/repo/Core App Tests - WebPlayer.postman_collection.json -e /tmp/testkube-tmp1897306484 --reporters cli,json --reporter-json-export /tmp/testkube-tmp4151526495.json","time":"2023-07-18T19:12:44.066856036Z"} 
{"type":"line","content":"🔬 Executing in directory : \n $ newman run /data/repo/Core App Tests - WebPlayer.postman_collection.json -e /tmp/testkube-tmp1897306484 --reporters cli,json --reporter-json-export /tmp/testkube-tmp4151526495.json","time":"2023-07-18T19:12:44.066872166Z"} 
{"type":"line","content":"newman\n\n","time":"2023-07-18T19:12:45.654031992Z"} 
{"type":"line","content":"Core App Tests - WebPlayer\n","time":"2023-07-18T19:12:45.654989975Z"} 
{"type":"line","content":"\n→ na.com client=testdb sign=testct1 company=41574150-b952-413b-898b-dc5336b4bd12\n","time":"2023-07-18T19:12:45.658607412Z"} 
{"type":"line","content":"  GET https://na.com/v6-wplt/?client=testdb\u0026sign=testct1\u0026company=41574150-b952-413b-898b-dc5336b4bd12 ","time":"2023-07-18T19:12:45.685011667Z"} 
{"type":"line","content":"[200 OK, 33.9kB, 326ms]\n","time":"2023-07-18T19:12:46.014247794Z"} 
{"type":"line","content":"  ✓  Status code is 200\n","time":"2023-07-18T19:12:46.076176552Z"}
{"type":"line","content":"\n┌─────────────────────────┬────────────────────┬───────────────────┐\n│                         │           executed │            failed │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              iterations │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│                requests │                  1 │   0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│            test-scripts │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│      prerequest-scripts │                  0 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              assertions │                  1 │ 0 │\n├─────────────────────────┴────────────────────┴───────────────────┤\n│ total run duration: 429ms                                        │\n├──────────────────────────────────────────────────────────────────┤\n│ total data received: 33.45kB (approx)                            │\n├──────────────────────────────────────────────────────────────────┤\n│ average response time: 326ms [min: 326ms, max: 326ms, s.d.: 0µs] │\n└──────────────────────────────────────────────────────────────────┘\n","time":"2023-07-18T19:12:46.104174132Z"} 
{"type":"line","content":"✅ Execution succeeded","time":"2023-07-18T19:12:46.114565626Z"} 
{"type":"line","content":"✅ Got Newman result successfully","time":"2023-07-18T19:12:46.126116248Z"} 
{"type":"line","content":"✅ Mapped Newman result successfully","time":"2023-07-18T19:12:46.126152021Z"} 
{"type":"result","result":{"status":"passed","output":"newman\n\nCore App Tests - WebPlayer\n\n→ core-eks-test.poppcore.co client=testdb sign=testct1 company=41574150-b952-413b-898b-dc5336b4bd12\n  GET https://na.com/v6-wplt/?client=testdb\u0026sign=testct1\u0026company=41574150-b952-413b-898b-dc5336b4bd12 [200 OK, 33.9kB, 326ms]\n  ✓  Status code is 200\n\n┌─────────────────────────┬────────────────────┬───────────────────┐\n│                         │           executed │            failed │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              iterations │                  1 │      0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│                requests │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│            test-scripts │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│      prerequest-scripts │                  0 │ 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              assertions │                  1 │                 0 │\n├─────────────────────────┴────────────────────┴───────────────────┤\n│ total run duration: 429ms                                        │\n├──────────────────────────────────────────────────────────────────┤\n│ total data received: 33.45kB (approx)                            │\n├──────────────────────────────────────────────────────────────────┤\n│ average response time: 326ms [min: 326ms, max: 326ms, s.d.: 0µs] │\n└──────────────────────────────────────────────────────────────────┘\n","outputType":"text/plain","steps":[{"name":"na.com client=testdb sign=testct1 company=41574150-b952-413b-898b-dc5336b4bd12","duration":"326ms","status":"passed","assertionResults":[{"name":"Status code is 200","status":"passed"}]}]},"time":"2023-07-18T19:12:46.12615853Z"}`)
		result, output, err := ParseContainerOutput(exampleOutput)

		assert.Len(t, output, 7305)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
	})

	t.Run("Output with failed result", func(t *testing.T) {
		t.Parallel()

		output := []byte(`{"type":"event","content":"running test [63c9606d7104b0fa0b7a45f1]"}
{"type":"event","content":"Running [ ./zap-api-scan.py [-t https://www.example.com/openapi.json -f openapi -c examples/zap-api.conf -d -D 5 -I -l INFO -n examples/context.config -S -T 60 -U anonymous -O https://www.example.com -z -config aaa=bbb -r api-test-report.html]]"}
{"type":"result","result":{"status":"failed","errorMessage":"could not start process: fork/exec ./zap-api-scan.py: no such file or directory"}}`)
		expectedOutput := `running test [63c9606d7104b0fa0b7a45f1]
Running [ ./zap-api-scan.py [-t https://www.example.com/openapi.json -f openapi -c examples/zap-api.conf -d -D 5 -I -l INFO -n examples/context.config -S -T 60 -U anonymous -O https://www.example.com -z -config aaa=bbb -r api-test-report.html]]
could not start process: fork/exec ./zap-api-scan.py: no such file or directory
`
		result, data, err := ParseContainerOutput(output)

		assert.Equal(t, expectedOutput, data)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "could not start process: fork/exec ./zap-api-scan.py: no such file or directory", result.ErrorMessage)
	})

	t.Run("Output with error", func(t *testing.T) {
		t.Parallel()

		output := []byte(`
{"type":"line","content":"🚚 Preparing test runner","time":"2023-01-19T15:22:25.867379429Z"}
{"type":"line","content":"🌍 Reading environment variables...","time":"2023-01-19T15:22:25.867927513Z"}
{"type":"line","content":"✅ Environment variables read successfully","time":"2023-01-19T15:22:25.867944763Z"}
{"type":"line","content":"RUNNER_ENDPOINT=\"testkube-minio-service-testkube:9000\"","time":"2023-01-19T15:22:25.867946929Z"}
{"type":"line","content":"RUNNER_ACCESSKEYID=\"********\"","time":"2023-01-19T15:22:25.867948804Z"}
{"type":"line","content":"RUNNER_SECRETACCESSKEY=\"********\"","time":"2023-01-19T15:22:25.867955263Z"}
{"type":"line","content":"RUNNER_REGION=\"\"","time":"2023-01-19T15:22:25.867962596Z"}
{"type":"line","content":"RUNNER_TOKEN=\"\"","time":"2023-01-19T15:22:25.867967971Z"}
{"type":"line","content":"RUNNER_SSL=false","time":"2023-01-19T15:22:25.867974013Z"}
{"type":"line","content":"RUNNER_SCRAPPERENABLED=\"true\"","time":"2023-01-19T15:22:25.867978888Z"}
{"type":"line","content":"RUNNER_GITUSERNAME=\"\"","time":"2023-01-19T15:22:25.867984179Z"}
{"type":"line","content":"RUNNER_GITTOKEN=\"\"","time":"2023-01-19T15:22:25.867986013Z"}
{"type":"line","content":"RUNNER_DATADIR=\"/data\"","time":"2023-01-19T15:22:25.867987596Z"}
{"type":"line","content":"RUNNER_CDEVENTS_TARGET=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_DASHBOARD_URI=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"line","content":"RUNNER_CLUSTERID=\"\"","time":"2023-01-17T15:29:17.921801596Z"}
{"type":"event","content":"running test [63c960287104b0fa0b7a45ef]","time":"2023-01-19T15:22:25.868132888Z"}
{"type":"line","content":"🚚 Preparing for test run","time":"2023-01-19T15:22:25.868161346Z"}
{"type":"line","content":"❌ can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-19T15:22:25.868183971Z"}
{"type":"error","content":"can't find branch or commit in params, repo:\u0026{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:\u003cnil\u003e TokenSecret:\u003cnil\u003e WorkingDir:}","time":"2023-01-19T15:22:25.868198429Z"}
`)
		expectedOutput := `
🚚 Preparing test runner
🌍 Reading environment variables...
✅ Environment variables read successfully
RUNNER_ENDPOINT="testkube-minio-service-testkube:9000"
RUNNER_ACCESSKEYID="********"
RUNNER_SECRETACCESSKEY="********"
RUNNER_REGION=""
RUNNER_TOKEN=""
RUNNER_SSL=false
RUNNER_SCRAPPERENABLED="true"
RUNNER_GITUSERNAME=""
RUNNER_GITTOKEN=""
RUNNER_DATADIR="/data"
RUNNER_CDEVENTS_TARGET=""
RUNNER_DASHBOARD_URI=""
RUNNER_CLUSTERID=""
running test [63c960287104b0fa0b7a45ef]
🚚 Preparing for test run
❌ can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}
`

		result, data, err := ParseContainerOutput(output)

		assert.Equal(t, expectedOutput, data)
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, "can't find branch or commit in params, repo:&{Type_:git-file Uri:https://github.com/kubeshop/testkube.git Branch: Commit: Path:test/cypress/executor-smoke/cypress-11 Username: Token: UsernameSecret:<nil> TokenSecret:<nil> WorkingDir:}", result.ErrorMessage)
	})

	t.Run("Output ignore empty result", func(t *testing.T) {
		t.Parallel()

		output := []byte(`
{"type":"result","result":{"status":"running"},"time":"2023-07-19T17:15:20.400138Z"}
{"type":"result","result":{"status":null},"time":"2023-07-19T17:15:38.313334598Z"}
`)
		expectedOutput := `
running

`

		result, data, err := ParseContainerOutput(output)

		assert.Equal(t, expectedOutput, data)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Output random json format", func(t *testing.T) {
		t.Parallel()

		output := []byte(`
{"level":"info","ts":1689786938.2338681,"caller":"scraper/filesystem_extractor.go:50","msg":"scraping artifacts in directory: /data/artifacts"}
{"level":"info","ts":1689786938.234103,"caller":"scraper/filesystem_extractor.go:96","msg":"creating artifacts tarball with 1 files"}
{"level":"info","ts":1689786938.234986,"caller":"scraper/minio_uploader.go:47","msg":"MinIO loader is uploading file","file":"artifacts.tar.gz","folder":"64b81a176ebd74e68253f73e","size":12181}	
`)
		expectedOutput := `
{"level":"info","ts":1689786938.2338681,"caller":"scraper/filesystem_extractor.go:50","msg":"scraping artifacts in directory: /data/artifacts"}
{"level":"info","ts":1689786938.234103,"caller":"scraper/filesystem_extractor.go:96","msg":"creating artifacts tarball with 1 files"}
{"level":"info","ts":1689786938.234986,"caller":"scraper/minio_uploader.go:47","msg":"MinIO loader is uploading file","file":"artifacts.tar.gz","folder":"64b81a176ebd74e68253f73e","size":12181}	
`
		result, data, err := ParseContainerOutput(output)

		assert.Equal(t, expectedOutput, data)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Output random text format", func(t *testing.T) {
		t.Parallel()

		output := []byte(`
[STARTED] Task without title.
[SUCCESS] Task without title.
[203:0719/171524.648215:ERROR:zygote_host_impl_linux.cc(263)] Failed to adjust OOM score of renderer with pid 374: Permission denied (13)
libva error: vaGetDriverNameByIndex() failed with unknown libva error, driver_name = (null)
[374:0719/171524.697752:ERROR:gpu_memory_buffer_support_x11.cc(44)] dri3 extension not supported.
[374:0719/171524.697752:ERROR:gpu_memory_buffer_support_x11.cc(44)] dri3 extension not supported.

====================================================================================================
`)
		expectedOutput := `
[STARTED] Task without title.
[SUCCESS] Task without title.
[203:0719/171524.648215:ERROR:zygote_host_impl_linux.cc(263)] Failed to adjust OOM score of renderer with pid 374: Permission denied (13)
libva error: vaGetDriverNameByIndex() failed with unknown libva error, driver_name = (null)
[374:0719/171524.697752:ERROR:gpu_memory_buffer_support_x11.cc(44)] dri3 extension not supported.
[374:0719/171524.697752:ERROR:gpu_memory_buffer_support_x11.cc(44)] dri3 extension not supported.

====================================================================================================
`
		result, data, err := ParseContainerOutput(output)

		assert.Equal(t, expectedOutput, data)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}
