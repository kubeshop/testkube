package v1

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// UploadFiles uploads files into the object store and uses them during execution
func (s TestkubeAPI) UploadFiles() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to upload file"

		parentName := c.FormValue("parentName")
		if parentName == "" {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("%s: wrong input: parentName cannot be empty", errPrefix))
		}
		parentType := c.FormValue("parentType")
		if parentType == "" {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("%s: wrong input: parentType cannot be empty", errPrefix))
		}
		filePath := c.FormValue("filePath")
		if filePath == "" {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("%s: wrong input: filePath cannot be empty", errPrefix))
		}

		bucketName := s.ArtifactsStorage.GetValidBucketName(parentType, parentName)
		file, err := c.FormFile("attachment")
		if err != nil {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("%s: unable to upload file: %w", errPrefix, err))
		}
		f, err := file.Open()
		if err != nil {
			return s.Error(c, fiber.StatusBadRequest, fmt.Errorf("%s: cannot read file: %w", errPrefix, err))
		}
		defer f.Close()

		err = s.ArtifactsStorage.UploadFile(c.Context(), bucketName, filePath, f, file.Size)
		if err != nil {
			return s.Error(c, fiber.StatusInternalServerError, fmt.Errorf("%s: could not save uploaded file: %w", errPrefix, err))
		}

		return c.JSON(fiber.StatusOK)
	}
}
