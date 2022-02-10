package v1

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	scriptsmapper "github.com/kubeshop/testkube/pkg/mapper/scripts"
	"github.com/kubeshop/testkube/pkg/secret"

	"github.com/kubeshop/testkube/pkg/jobs"
	"k8s.io/apimachinery/pkg/api/errors"
)

// GetTestHandler is method for getting an existing script
func (s TestkubeAPI) GetTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		crScript, err := s.ScriptsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsmapper.MapScriptCRToAPI(*crScript)
		return c.JSON(scripts)
	}
}

// ListTestsHandler is a method for getting list of all available scripts
func (s TestkubeAPI) ListTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")

		rawTags := c.Query("tags")
		var tags []string
		if rawTags != "" {
			tags = strings.Split(rawTags, ",")
		}

		// TODO filters looks messy need to introduce some common Filter object for Kubernetes query for List like objects
		crScripts, err := s.ScriptsClient.List(namespace, tags)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		search := c.Query("textSearch")
		if search != "" {
			// filter items array
			for i := len(crScripts.Items) - 1; i >= 0; i-- {
				if !strings.Contains(crScripts.Items[i].Name, search) {
					crScripts.Items = append(crScripts.Items[:i], crScripts.Items[i+1:]...)
				}
			}
		}

		scriptType := c.Query("type")
		if scriptType != "" {
			// filter items array
			for i := len(crScripts.Items) - 1; i >= 0; i-- {
				if !strings.Contains(crScripts.Items[i].Spec.Type_, scriptType) {
					crScripts.Items = append(crScripts.Items[:i], crScripts.Items[i+1:]...)
				}
			}
		}

		scripts := scriptsmapper.MapScriptListKubeToAPI(*crScripts)

		return c.JSON(scripts)
	}
}

// CreateTestHandler creates new script CR based on script content
func (s TestkubeAPI) CreateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.TestUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("creating script", "request", request)

		scriptSpec := scriptsmapper.MapScriptToScriptSpec(request)
		script, err := s.ScriptsClient.Create(scriptSpec)

		s.Metrics.IncCreateTest(script.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		stringData := GetSecretsStringData(request.Content)
		if err = s.SecretClient.Create(secret.GetMetadataName(request.Name), request.Namespace, stringData); err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(script)
	}
}

// UpdateTestHandler updates an existing script CR based on script content
func (s TestkubeAPI) UpdateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.TestUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("updating script", "request", request)

		// we need to get resource first and load its metadata.ResourceVersion
		script, err := s.ScriptsClient.Get(request.Namespace, request.Name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		// map script but load spec only to not override metadata.ResourceVersion
		scriptSpec := scriptsmapper.MapScriptToScriptSpec(request)
		script.Spec = scriptSpec.Spec
		script, err = s.ScriptsClient.Update(script)

		s.Metrics.IncUpdateTest(script.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		// update secrets for scipt
		stringData := GetSecretsStringData(request.Content)
		if err = s.SecretClient.Apply(secret.GetMetadataName(request.Name), request.Namespace, stringData); err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(script)
	}
}

// DeleteTestHandler is a method for deleting a script with id
func (s TestkubeAPI) DeleteTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		err := s.ScriptsClient.Delete(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// delete secrets for script
		if err = s.SecretClient.Delete(secret.GetMetadataName(name), namespace); err != nil {
			if errors.IsNotFound(err) {
				return c.SendStatus(fiber.StatusNoContent)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// DeleteTestsHandler for deleting all scripts
func (s TestkubeAPI) DeleteTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		err := s.ScriptsClient.DeleteAll(namespace)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// delete all secrets for scripts
		if err = s.SecretClient.DeleteAll(namespace); err != nil {
			if errors.IsNotFound(err) {
				return c.SendStatus(fiber.StatusNoContent)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func GetSecretsStringData(content *testkube.TestContent) map[string]string {
	// create secrets for script
	stringData := map[string]string{jobs.GitUsernameSecretName: "", jobs.GitTokenSecretName: ""}
	if content != nil && content.Repository != nil {
		stringData[jobs.GitUsernameSecretName] = content.Repository.Username
		stringData[jobs.GitTokenSecretName] = content.Repository.Token
	}

	return stringData
}
