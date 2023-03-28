package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/contrib/executor/curl/pkg/curl"
	"github.com/kubeshop/testkube/contrib/executor/k6/pkg/k6detector"
	"github.com/kubeshop/testkube/contrib/executor/postman/pkg/postman"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	exampleValidContent = `{ "info": { "_postman_id": "3d9a6be2-bd3e-4cf7-89ca-354103aab4a7", "name": "Testkube", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" }, "item": [ { "name": "Health", "event": [ { "listen": "test", "script": { "exec": [ "pm.test(\"Status code is 200\", function () {", "    pm.response.to.have.status(200);", "});" ], "type": "text/javascript" } } ], "request": { "method": "GET", "header": [], "url": { "raw": "{{URI}}/health", "host": [ "{{URI}}" ], "path": [ "health" ] } }, "response": [] } ] } `
)

func TestDetectorDetect(t *testing.T) {

	t.Run("Detect test returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(curl.Detector{})
		detector.Add(postman.Detector{})
		detector.Add(k6detector.Detector{})

		name, found := detector.Detect("postman_collection.json", client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleValidContent),
		})

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "postman/collection", name)
	})

}

func TestDetectorDetectTestName(t *testing.T) {

	t.Run("Detect test name returns success", func(t *testing.T) {

		detector := postman.Detector{}

		name, found := detector.IsTestName("test.postman_collection.json")

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "test", name)
	})

	t.Run("Detect test name returns failure", func(t *testing.T) {

		detector := postman.Detector{}

		name, found := detector.IsTestName("test.json")

		assert.False(t, found, "detector should not find test type")
		assert.Empty(t, name)
	})

}

func TestDetectorDetectEnvName(t *testing.T) {

	t.Run("Detect env name returns success", func(t *testing.T) {

		detector := postman.Detector{}

		name, envName, found := detector.IsEnvName("test.prod.postman_environment.json")

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "test", name)
		assert.Equal(t, "prod", envName)
	})

	t.Run("Detect env name returns failure", func(t *testing.T) {

		detector := postman.Detector{}

		name, envName, found := detector.IsEnvName("test.prod.json")

		assert.False(t, found, "detector should not find test type")
		assert.Empty(t, name)
		assert.Empty(t, envName)
	})

}

func TestDetectorDetectSecretEnvName(t *testing.T) {

	t.Run("Detect secret env name returns success", func(t *testing.T) {

		detector := postman.Detector{}

		name, envName, found := detector.IsSecretEnvName("test.dev.postman_secret_environment.json")

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "test", name)
		assert.Equal(t, "dev", envName)
	})

	t.Run("Detect secret env name returns failure", func(t *testing.T) {

		detector := postman.Detector{}

		name, envName, found := detector.IsSecretEnvName("test.dev.json")

		assert.False(t, found, "detector should not find test type")
		assert.Empty(t, name)
		assert.Empty(t, envName)
	})

}

func TestDetectorGetAdapter(t *testing.T) {

	t.Run("Get adapter returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(curl.Detector{})
		detector.Add(postman.Detector{})
		detector.Add(k6.Detector{})

		adapter := detector.GetAdapter("postman/collection")

		assert.NotNil(t, adapter)
	})

}
