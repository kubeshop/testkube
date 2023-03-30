package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/contrib/executor/artillery/pkg/artillery"
	"github.com/kubeshop/testkube/contrib/executor/curl/pkg/curl"
	"github.com/kubeshop/testkube/contrib/executor/jmeter/pkg/jmeter"
	"github.com/kubeshop/testkube/contrib/executor/k6/pkg/k6detector"
	"github.com/kubeshop/testkube/contrib/executor/postman/pkg/postman"
	"github.com/kubeshop/testkube/contrib/executor/soapui/pkg/soapui"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestDetectorDetect(t *testing.T) {

	t.Run("Detect test returns success", func(t *testing.T) {

		detector := Detector{Adapters: make(map[string]Adapter, 0)}
		detector.Add(curl.Detector{})
		detector.Add(postman.Detector{})
		detector.Add(k6detector.Detector{})

		name, found := detector.Detect("postman_collection.json", client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(examplePostmanContent),
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
		detector.Add(k6detector.Detector{})

		adapter := detector.GetAdapter("postman/collection")

		assert.NotNil(t, adapter)
	})

}

func TestDifferentTestTypes(t *testing.T) {
	t.Run("Detect postman collection", func(t *testing.T) {
		detector := NewDefaultDetector()
		name, found := detector.Detect("postman_collection.json", client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(examplePostmanContent),
		})

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, postman.PostmanCollectionType, name)
	})

	t.Run("Detect artillery test", func(t *testing.T) {
		detector := NewDefaultDetector()
		name, found := detector.Detect(exampleArtilleryFilename, client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleArtilleryContent),
		})

		assert.True(t, found, "detector should find artillery/test")
		assert.Equal(t, artillery.Type, name)
	})

	t.Run("Detect cURL test", func(t *testing.T) {
		detector := NewDefaultDetector()
		name, found := detector.Detect(exampleCurlFilename, client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleCurlContent),
		})

		assert.True(t, found, "detector should find curl/test")
		assert.Equal(t, curl.Type, name)
	})

	t.Run("Detect jmeter test", func(t *testing.T) {
		detector := NewDefaultDetector()
		name, found := detector.Detect(exampleJMeterFilename, client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleJMeterContent),
		})

		assert.True(t, found, "detector should find jmeter/test")
		assert.Equal(t, jmeter.Type, name)
	})

	t.Run("Detect k6 test", func(t *testing.T) {
		detector := NewDefaultDetector()
		name, found := detector.Detect(exampleK6Filename, client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleK6Content),
		})

		assert.True(t, found, "detector should find k6/test")
		assert.Equal(t, k6detector.Type, name)
	})

	t.Run("Detect soapui test", func(t *testing.T) {
		detector := NewDefaultDetector()
		name, found := detector.Detect(exampleSoapUIFilename, client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleSoapUIContent),
		})

		assert.True(t, found, "detector should find soapui/test")
		assert.Equal(t, soapui.Type, name)
	})
}
