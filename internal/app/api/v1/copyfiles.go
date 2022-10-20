package v1

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// UploadCopyFiles uploads files into the object store and uses them during execution
func (s TestkubeAPI) UploadCopyFiles() fiber.Handler {
	return func(c *fiber.Ctx) error {
		parentID := c.FormValue("parentID")
		if parentID == "" {
			return s.Error(c, fiber.StatusBadRequest, errors.New("parentID cannot be empty"))
		}
		parentType := c.FormValue("parentType")
		if parentType == "" {
			return s.Error(c, fiber.StatusBadRequest, errors.New("parentType cannot be empty"))
		}
		filePath := c.FormValue("filePath")
		if filePath == "" {
			return s.Error(c, fiber.StatusBadRequest, errors.New("filePath cannot be empty"))
		}

		bucketName := getBucketName(parentType, parentID)
		file, err := c.FormFile("attachment")
		if err != nil {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("unable to upload file: %w", err))
		}
		f, err := file.Open()
		if err != nil {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("cannot read file: %d", err))
		}
		err = s.Storage.SaveCopyFile(bucketName, filePath, f, file.Size)
		if err != nil {
			return s.Error(c, fiber.StatusInternalServerError, fmt.Errorf("could not save copy file: %w", err))
		}

		return c.JSON(fiber.StatusOK)
	}
}

func getBucketName(parentType string, parentID string) string {
	return fmt.Sprintf("%s-%s", parentType, parentID)
}
