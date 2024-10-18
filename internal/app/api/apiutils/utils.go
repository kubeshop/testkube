package apiutils

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/problem"
	"github.com/kubeshop/testkube/pkg/secretmanager"
)

const (
	// mediaTypeYAML is yaml media type
	mediaTypeYAML = "text/yaml"
)

func SendLegacyCRDs(c *fiber.Ctx, data string, err error) error {
	if err != nil {
		return SendError(c, http.StatusBadRequest, fmt.Errorf("could not build CRD: %w", err))
	}

	c.Context().SetContentType(mediaTypeYAML)
	return c.SendString(data)
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

// SendWarn writes RFC-7807 json problem to response
func SendWarn(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	c.Status(status)
	c.Response().Header.Set("Content-Type", "application/problem+json")
	log.DefaultLogger.Warnw(err.Error(), "status", status)
	pr := problem.New(status, getProblemMessage(err, context))
	return c.JSON(pr)
}

// SendError writes RFC-7807 json problem to response
func SendError(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	c.Status(status)
	c.Response().Header.Set("Content-Type", "application/problem+json")
	log.DefaultLogger.Errorw(err.Error(), "status", status)
	pr := problem.New(status, getProblemMessage(err, context))
	return c.JSON(pr)
}

// getProblemMessage creates new JSON based problem message and returns it as string
func getProblemMessage(err error, context ...interface{}) string {
	message := err.Error()
	if len(context) > 0 {
		b, err := json.Marshal(context[0])
		if err == nil {
			message += ", context: " + string(b)
		}
	}

	return message
}
