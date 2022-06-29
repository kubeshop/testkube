package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
)

// Gettable is an interface of gettable objects
type Gettable interface {
	testkube.Test | testkube.TestSuite | testkube.ExecutorCreateRequest | testkube.Webhook
}

func (s TestkubeAPI) getCRD(c *fiber.Ctx, tmpl crd.Template, item any) error {
	yaml, err := crd.ExecuteTemplate(tmpl, item)
	if err != nil {
		return s.Error(c, http.StatusBadRequest, err)
	}

	c.Context().SetContentType(mediaTypeYAML)
	return c.SendString(yaml)
}

func (s TestkubeAPI) getCRDs(c *fiber.Ctx, data string, err error) error {
	if err != nil {
		return s.Error(c, http.StatusBadRequest, err)
	}

	c.Context().SetContentType(mediaTypeYAML)
	return c.SendString(data)
}

// GenerateCRDs generates CRDs for Testkube models
func GenerateCRDs[G Gettable](tmpl crd.Template, items []G) (string, error) {
	data := ""
	firstEntry := true
	for _, item := range items {
		crd, err := crd.ExecuteTemplate(tmpl, item)
		if err != nil {
			return "", err
		}

		if !firstEntry {
			data += "\n---\n"
		} else {
			firstEntry = false
		}

		data += crd
	}

	return data, nil
}
