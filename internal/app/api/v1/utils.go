package v1

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/problem"
	"github.com/kubeshop/testkube/pkg/secretmanager"
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
	b, err := common.SerializeCRDs(crds, common.SerializeOptions{
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

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mongo.ErrNoDocuments) || k8serrors.IsNotFound(err) || errors.Is(err, secretmanager.ErrNotFound) {
		return true
	}
	if e, ok := status.FromError(err); ok {
		return e.Code() == codes.NotFound
	}
	return false
}

// Warn writes RFC-7807 json problem to response
func (s *TestkubeAPI) Warn(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	c.Status(status)
	c.Response().Header.Set("Content-Type", "application/problem+json")
	s.Log.Warnw(err.Error(), "status", status)
	pr := problem.New(status, s.getProblemMessage(err, context))
	return c.JSON(pr)
}

// Error writes RFC-7807 json problem to response
func (s *TestkubeAPI) Error(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	c.Status(status)
	c.Response().Header.Set("Content-Type", "application/problem+json")
	s.Log.Errorw(err.Error(), "status", status)
	pr := problem.New(status, s.getProblemMessage(err, context))
	return c.JSON(pr)
}

// getProblemMessage creates new JSON based problem message and returns it as string
func (s *TestkubeAPI) getProblemMessage(err error, context ...interface{}) string {
	message := err.Error()
	if len(context) > 0 {
		b, err := json.Marshal(context[0])
		if err == nil {
			message += ", context: " + string(b)
		}
	}

	return message
}
