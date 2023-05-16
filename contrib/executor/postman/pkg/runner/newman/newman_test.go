package newman

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/kubeshop/testkube/pkg/envs"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// TestRun runs newman instance on top of example collection
// creates temporary server and check if call to the server was done from newman
func TestRun_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()
	// given
	tempDir, err := os.MkdirTemp("", "*")
	assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	runner, err := NewNewmanRunner(context.Background(), envs.Params{DataDir: tempDir})
	assert.NoError(t, err)

	// and test server for getting newman responses
	requestCompleted := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCompleted = true
	}))

	defer ts.Close()

	parts := strings.Split(ts.URL, ":")
	port := parts[2]

	err = os.WriteFile(filepath.Join(tempDir, "test-content"), []byte(fmt.Sprintf(exampleCollection, port, port)), 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write postman runner test content file")
	}

	execution := testkube.Execution{
		Content: testkube.NewStringTestContent(""),
		Command: []string{"newman"},
		Args: []string{
			"run",
			"<runPath>",
			"-e",
			"<envFile>",
			"--reporters",
			"cli,json",
			"--reporter-json-export",
			"<reportFile>",
		},
	}

	ctx := context.Background()

	// when
	result, err := runner.Run(ctx, execution)

	// then
	assert.NoError(t, err)
	assert.Empty(t, result.ErrorMessage)
	assert.Contains(t, result.Output, "Successful GET request")
	assert.Equal(t, requestCompleted, true)
}

const exampleCollection = `
{
	"info": {
		"_postman_id": "3d9a6be2-bd3e-4cf7-89ca-354103aab4a7",
		"name": "testkube",
		"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	},
	"item": [
		{
			"name": "Test",
			"event": [
				{
					"listen": "test",
					"script": {
						"exec": [
							"    pm.test(\"Successful GET request\", function () {",
							"        pm.expect(pm.response.code).to.be.oneOf([200, 201, 202]);",
							"    });"
						],
						"type": "text/javascript"
					}
				}
			],
			"request": {
				"method": "GET",
				"header": [],
				"url": {
					"raw": "http://127.0.0.1:%s",
					"protocol": "http",
					"host": [
						"127",
						"0",
						"0",
						"1"
					],
					"port": "%s"
	
				},
				"host": ["localhost"]
			},
			"response": []
		}
	]
}
`
