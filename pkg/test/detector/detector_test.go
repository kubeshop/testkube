package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestDetectorDetect(t *testing.T) {

	t.Run("Detect test returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		name, found := detector.Detect(client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleValidContent),
		})

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "postman/collection", name)
	})

}

func TestDetectorDetectTestName(t *testing.T) {

	t.Run("Detect test name returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		name, testType, found := detector.DetectTestName("test.postman_collection.json")

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "test", name)
		assert.Equal(t, "postman/collection", testType)
	})

	t.Run("Detect test name returns failure", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		name, testType, found := detector.DetectTestName("test.json")

		assert.False(t, found, "detector should not find test type")
		assert.Empty(t, name)
		assert.Empty(t, testType)
	})

}

func TestDetectorDetectEnvName(t *testing.T) {

	t.Run("Detect env name returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		name, envName, testType, found := detector.DetectEnvName("test.prod.postman_environment.json")

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "test", name)
		assert.Equal(t, "prod", envName)
		assert.Equal(t, "postman/collection", testType)
	})

	t.Run("Detect env name returns failure", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		name, envName, testType, found := detector.DetectEnvName("test.prod.json")

		assert.False(t, found, "detector should not find test type")
		assert.Empty(t, name)
		assert.Empty(t, envName)
		assert.Empty(t, testType)
	})

}

func TestDetectorDetectSecretEnvName(t *testing.T) {

	t.Run("Detect secret env name returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		name, envName, testType, found := detector.DetectSecretEnvName("test.dev.postman_secret_environment.json")

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "test", name)
		assert.Equal(t, "dev", envName)
		assert.Equal(t, "postman/collection", testType)
	})

	t.Run("Detect secret env name returns failure", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		name, envName, testType, found := detector.DetectSecretEnvName("test.dev.json")

		assert.False(t, found, "detector should not find test type")
		assert.Empty(t, name)
		assert.Empty(t, envName)
		assert.Empty(t, testType)
	})

}

func TestDetectorGetAdapter(t *testing.T) {

	t.Run("Get adapter returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})
		detector.Add(K6Adapter{})

		adapter := detector.GetAdapter("postman/collection")

		assert.NotNil(t, adapter)
	})

}
