package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	failedCSV = `timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect
1667462863619,441,HTTP Request,200,OK,Thread Group 1-1,text,true,,66428,109,1,1,https://testkube.io,385,0,297
1667462929945,390,HTTP Request,200,OK,Thread Group 1-1,text,false,Test failed: text expected to contain /SOME_NONExisting_String/,66428,109,1,1,https://testkube.io,339,0,249
1667462945507,344,HTTP Request,200,OK,Thread Group 1-1,text,true,,66428,109,1,1,https://testkube.io,294,0,207
`

	successCSV = `timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect
1667463814102,382,HTTP Request,200,OK,Thread Group 1-1,text,true,,66428,109,1,1,https://testkube.io,326,0,235
1667463836936,365,HTTP Request,200,OK,Thread Group 1-1,text,true,,66428,109,1,1,https://testkube.io,309,0,222
1667463838447,362,HTTP Request,200,OK,Thread Group 1-1,text,true,,66428,109,1,1,https://testkube.io,309,0,219
`

	errorCSV = `timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect
`
)

func TestParseCSV(t *testing.T) {
	t.Parallel()

	t.Run("parse CSV error test", func(t *testing.T) {
		t.Parallel()

		results, err := parseCSV(strings.NewReader(errorCSV))

		assert.NoError(t, err)
		assert.Nil(t, results.Results)
	})

	t.Run("parse CSV failed test", func(t *testing.T) {
		t.Parallel()

		results, err := parseCSV(strings.NewReader(failedCSV))

		assert.NoError(t, err)
		assert.True(t, results.HasError)
		assert.Equal(t, 3, len(results.Results))
		assert.Equal(t, "Test failed: text expected to contain /SOME_NONExisting_String/", results.Results[1].Error)
		assert.Equal(t, "Test failed: text expected to contain /SOME_NONExisting_String/", results.LastErrorMessage)
	})

	t.Run("parse CSV success test", func(t *testing.T) {
		t.Parallel()

		results, err := parseCSV(strings.NewReader(successCSV))

		assert.NoError(t, err)
		assert.False(t, results.HasError)
		assert.Equal(t, "200", results.Results[0].ResponseCode)
		assert.Equal(t, time.Millisecond*382, results.Results[0].Duration)
		assert.Equal(t, "HTTP Request", results.Results[0].Label)
	})
}
