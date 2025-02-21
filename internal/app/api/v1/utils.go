package v1

import (
	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/internal/crdcommon"
)

func ExpectsYAML(c *fiber.Ctx) bool {
	accept := c.Accepts(mediaTypeJSON, mediaTypeYAML, mediaTypeYAMLAlt)
	return accept == mediaTypeYAML || accept == mediaTypeYAMLAlt || c.Query("_yaml") == "true"
}

func HasYAML(c *fiber.Ctx) bool {
	contentType := string(c.Request().Header.ContentType())
	return contentType == mediaTypeYAML || contentType == mediaTypeYAMLAlt
}

func SendResourceList[T interface{}, U interface{}](c *fiber.Ctx, kind string, groupVersion schema.GroupVersion, jsonMapper func(T) U, data ...T) error {
	if ExpectsYAML(c) {
		return SendCRDs(c, kind, groupVersion, data...)
	}
	result := make([]U, len(data))
	for i, item := range data {
		result[i] = jsonMapper(item)
	}
	return c.JSON(result)
}

func SendResource[T interface{}, U interface{}](c *fiber.Ctx, kind string, groupVersion schema.GroupVersion, jsonMapper func(T) U, data T) error {
	if ExpectsYAML(c) {
		return SendCRDs(c, kind, groupVersion, data)
	}
	return c.JSON(jsonMapper(data))
}

func SendCRDs[T interface{}](c *fiber.Ctx, kind string, groupVersion schema.GroupVersion, crds ...T) error {
	b, err := crdcommon.SerializeCRDs(crds, crdcommon.SerializeOptions{
		OmitCreationTimestamp: true,
		CleanMeta:             true,
		Kind:                  kind,
		GroupVersion:          &groupVersion,
	})
	if err != nil {
		return err
	}
	c.Context().SetContentType(mediaTypeYAML)
	return c.Send(b)
}
