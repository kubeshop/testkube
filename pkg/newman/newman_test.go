package newman

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const exampleCollection = `
{
	"info": {
		"_postman_id": "3d9a6be2-bd3e-4cf7-89ca-354103aab4a7",
		"name": "KubeTest",
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

func TestRunCollection(t *testing.T) {
	// given test server
	requestCompleted := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCompleted = true
	}))
	defer ts.Close()

	// and example content
	parts := strings.Split(ts.URL, ":")
	port := parts[2]
	buffer := strings.NewReader(fmt.Sprintf(exampleCollection, port, port))
	// and runner instance
	runner := &Runner{}

	// when
	result, err := runner.RunCollection(buffer)

	// then
	assert.NoError(t, err)
	assert.Contains(t, string(result.Output), "Successful GET request")
	assert.Equal(t, requestCompleted, true)

}

func TestRunner_SaveToTmpFile(t *testing.T) {
	t.Run("saves content to temporary file and return file path", func(t *testing.T) {
		// given
		const content = "test content"
		buffer := strings.NewReader(content)

		runner := Runner{}

		// when
		path, err := runner.SaveToTmpFile(buffer)

		// then
		assert.NoError(t, err)
		defer os.Remove(path) // clean up

		contentBytes, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}

		assert.Equal(t, content, string(contentBytes))
	})
}
