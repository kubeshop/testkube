package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/crd"
)

func (s TestkubeAPI) getCRD(c *fiber.Ctx, tmpl crd.Template, item any) error {
	yaml, err := crd.ExecuteTemplate(tmpl, item)
	if err != nil {
		return s.Error(c, http.StatusBadRequest, err)
	}

	c.Context().SetContentType(mediaTypeYAML)
	return c.SendString(yaml)
}

func (s TestkubeAPI) getCRDs(c *fiber.Ctx, tmpl crd.Template, items []any) error {
	data := ""
	firstEntry := true
	for _, item := range items {
		crd, err := crd.ExecuteTemplate(tmpl, item)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if !firstEntry {
			data += "\n---\n"
		} else {
			firstEntry = false
		}

		data += crd
	}

	c.Context().SetContentType(mediaTypeYAML)
	return c.SendString(data)
}
