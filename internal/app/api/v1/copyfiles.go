package v1

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// UploadCopyFiles uploads files into the object store and uses them during execution
func (s TestkubeAPI) UploadCopyFiles() fiber.Handler {
	return func(c *fiber.Ctx) error {
		fileName := c.Params("filename")
		bucketName := getBucketName(c.OriginalURL(), c.Params("id"))

		fileSize := c.Request().Header.ContentLength()
		if fileSize <= 0 {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("invalid content length: %d", fileSize))
		}

		err := s.Storage.SaveCopyFile(bucketName, fileName, c.Context().RequestBodyStream(), int64(fileSize))
		if err != nil {
			return s.Error(c, fiber.StatusInternalServerError, fmt.Errorf("could not save copy file: %w", err))
		}

		return c.JSON(fiber.StatusOK)
	}
}

func getBucketName(url string, ownerID string) string {
	if strings.Contains(url, "test-suites") {
		return fmt.Sprintf("test-suite-%s", ownerID)
	}
	if strings.Contains(url, "tests") {
		return fmt.Sprintf("test-%s", ownerID)
	}
	if strings.Contains(url, "executions") {
		return fmt.Sprintf("execution-%s", ownerID)
	}
	return ""
}
