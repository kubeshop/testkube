package postman

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	exampleValidContent                    = `{ "info": { "_postman_id": "3d9a6be2-bd3e-4cf7-89ca-354103aab4a7", "name": "Testkube", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" }, "item": [ { "name": "Health", "event": [ { "listen": "test", "script": { "exec": [ "pm.test(\"Status code is 200\", function () {", "    pm.response.to.have.status(200);", "});" ], "type": "text/javascript" } } ], "request": { "method": "GET", "header": [], "url": { "raw": "{{URI}}/health", "host": [ "{{URI}}" ], "path": [ "health" ] } }, "response": [] } ] } `
	exampleInvalidContent                  = `{"some":"json content"}`
	exampleInvalidJSONContent              = `some non json content`
	exampleValidSecretEnvironmentContent   = `{"id": "f8a038bf-3766-4424-94ee-381a69f55b9a", "name": "Testing env", "values": [{"key": "secenv1", "value": "var-secrets=homepage", "enabled": true}, {"key": "secenv2", "value": "var-secrets=apikey", "enabled": false}], "_postman_variable_scope": "environment", "_postman_exported_at": "2022-04-08T04:47:42.590Z", "_postman_exported_using": "Postman/9.14.14"}`
	exampleInValidSecretEnvironmentContent = `{"id": "f8a038bf-3766-4424-94ee-381a69f55b9a", "name": "Testing env", "values": [{"key": "secenv1", "value": "homepage", "enabled": true}], "_postman_variable_scope": "environment", "_postman_exported_at": "2022-04-08T04:47:42.590Z", "_postman_exported_using": "Postman/9.14.14"}`
)

func TestPostmanCollectionAdapterIs(t *testing.T) {

	t.Run("Is return true when valid content", func(t *testing.T) {
		detector := Detector{}
		name, is := detector.Is(client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleValidContent),
		})

		assert.Equal(t, "postman/collection", name)
		assert.True(t, is, "content should be of postman type")
	})

	t.Run("Is return false in case of invalid JSON content", func(t *testing.T) {
		detector := Detector{}
		name, is := detector.Is(client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleInvalidContent),
		})

		assert.Empty(t, name)
		assert.False(t, is, "content should not be of postman type")
	})

	t.Run("Is return false in case of content which is not JSON ", func(t *testing.T) {
		detector := Detector{}
		name, is := detector.Is(client.UpsertTestOptions{
			Content: testkube.NewStringTestContent(exampleInvalidJSONContent),
		})

		assert.Empty(t, name)
		assert.False(t, is, "content should not be of postman type")
	})
}

func TestPostmanCollectionAdapterIsTestName(t *testing.T) {

	t.Run("Is test name returns true when filename is valid", func(t *testing.T) {
		detector := Detector{}
		name, is := detector.IsTestName("test.postman_collection.json")

		assert.Equal(t, "test", name)
		assert.True(t, is, "filename should be valid test name")
	})

	t.Run("Is test name returns false when filename is invalid", func(t *testing.T) {
		detector := Detector{}
		name, is := detector.IsTestName("test.json")

		assert.Empty(t, name)
		assert.False(t, is, "filename should not be valid test name")
	})

}

func TestPostmanCollectionAdapterIsEnvName(t *testing.T) {

	t.Run("Is env name returns true when filename is valid", func(t *testing.T) {
		detector := Detector{}
		name, envName, is := detector.IsEnvName("test.prod.postman_environment.json")

		assert.Equal(t, "test", name)
		assert.Equal(t, "prod", envName)
		assert.True(t, is, "filename should be valid env name")
	})

	t.Run("Is env name returns false when filename is invalid", func(t *testing.T) {
		detector := Detector{}
		name, envName, is := detector.IsEnvName("test.dev.json")

		assert.Empty(t, name)
		assert.Empty(t, envName)
		assert.False(t, is, "filename should not be valid env name")
	})

	t.Run("Is env name returns false when filename doesn't contain env", func(t *testing.T) {
		detector := Detector{}
		name, envName, is := detector.IsEnvName("test.postman_environment.json")

		assert.Empty(t, name)
		assert.Empty(t, envName)
		assert.False(t, is, "filename should not be valid env name")
	})

}

func TestPostmanCollectionAdapterIsSecretEnvName(t *testing.T) {

	t.Run("Is secret env name returns true when filename is valid", func(t *testing.T) {
		detector := Detector{}
		name, envName, is := detector.IsSecretEnvName("test.prod.postman_secret_environment.json")

		assert.Equal(t, "test", name)
		assert.Equal(t, "prod", envName)
		assert.True(t, is, "filename should be valid secret env name")
	})

	t.Run("Is secret env name returns false when filename is invalid", func(t *testing.T) {
		detector := Detector{}
		name, envName, is := detector.IsSecretEnvName("test.dev.json")

		assert.Empty(t, name)
		assert.Empty(t, envName)
		assert.False(t, is, "filename should not be valid secret env name")
	})

	t.Run("Is secret env name returns false when filename doesn't contain env", func(t *testing.T) {
		detector := Detector{}
		name, envName, is := detector.IsEnvName("test.postman_secret_environment.json")

		assert.Empty(t, name)
		assert.Empty(t, envName)
		assert.False(t, is, "filename should not be valid secret env name")
	})

}

func TestPostmanCollectionAdapterGetSecretVariables(t *testing.T) {

	t.Run("Get secret variables returns enabled secret variables", func(t *testing.T) {
		detector := Detector{}
		variables, err := detector.GetSecretVariables(exampleValidSecretEnvironmentContent)

		assert.Equal(t, variables, map[string]testkube.Variable{"secenv1": testkube.NewSecretVariableReference("secenv1", "var-secrets", "homepage")})
		assert.NoError(t, err)
	})

	t.Run("Get secret variables returns no secret variable", func(t *testing.T) {
		detector := Detector{}
		variables, err := detector.GetSecretVariables(exampleInValidSecretEnvironmentContent)

		assert.Equal(t, variables, map[string]testkube.Variable{})
		assert.NoError(t, err)
	})

	t.Run("Get secret variables returns error for invalid json", func(t *testing.T) {
		detector := Detector{}
		variables, err := detector.GetSecretVariables(exampleInvalidJSONContent)

		assert.Nil(t, variables)
		assert.Error(t, err)
	})
}
