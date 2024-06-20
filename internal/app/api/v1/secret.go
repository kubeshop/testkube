package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/secrets"
)

// ListSecretsHandler list secrets and keys
func (s *TestkubeAPI) ListSecretsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list secrets"

		all, err := strconv.ParseBool(c.Query("all", "false"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse all parameter: %s", errPrefix, err))
		}

		namespace := c.Query("namespace")
		if namespace == "" {
			namespace = s.Namespace
		}

		selector := "createdBy=testkube"
		if all && s.enableSecretsEndpoint {
			selector = ""
		}
		list, err := s.Clientset.CoreV1().Secrets(namespace).List(c.Context(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list secrets: %s", errPrefix, err))
		}

		results := make([]testkube.Secret, len(list.Items))
		for i, secret := range list.Items {
			results[i] = secrets.MapSecretKubeToAPI(&secret)
		}

		return c.JSON(results)
	}
}

func (s *TestkubeAPI) CreateSecretHandler() fiber.Handler {
	errPrefix := "failed to create secret"
	return func(c *fiber.Ctx) (err error) {
		// Deserialize resource
		var input *testkube.SecretInput
		err = c.BodyParser(&input)
		if err != nil {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		// Validate resource
		if input == nil || input.Name == "" {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("name is required"))
		}
		if len(input.Data) == 0 {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("data should not be empty"))
		}

		// Apply defaults
		if input.Namespace == "" {
			input.Namespace = s.Namespace
		}
		if input.Labels == nil {
			input.Labels = map[string]string{}
		}
		input.Labels["createdBy"] = "testkube"
		owner := secrets.MapSecretOwnerAPIToKube(input.Owner)
		if owner == "" {
			input.Labels["testkubeOwner"] = owner
		} else {
			delete(input.Labels, "testkubeOwner")
		}

		// Create the resource
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: input.Name, Labels: input.Labels},
			StringData: input.Data,
		}
		secret, err = s.Clientset.CoreV1().Secrets(input.Namespace).Create(c.Context(), secret, metav1.CreateOptions{})
		if err != nil {
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		return c.JSON(secret)
	}
}

func (s *TestkubeAPI) DeleteSecretHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete secret '%s'", name)

		namespace := c.Query("namespace")
		if namespace == "" {
			namespace = s.Namespace
		}

		// Get the secret details
		secret, err := s.Clientset.CoreV1().Secrets(namespace).Get(c.Context(), name, metav1.GetOptions{})
		if err != nil {
			if IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get secret: %s", errPrefix, err))
		}

		// Disallow when it is not controlled by Testkube
		if secret.Labels["createdBy"] != "testkube" {
			if s.enableSecretsEndpoint {
				return s.Error(c, http.StatusForbidden, fmt.Errorf("%s: secret is not controlled by Testkube", errPrefix))
			} else {
				// Make it the same as when it's actually not found, to avoid blind search
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
			}
		}

		// Delete the secret
		err = s.Clientset.CoreV1().Secrets(namespace).Delete(c.Context(), name, metav1.DeleteOptions{
			GracePeriodSeconds: common.Ptr(int64(0)),
			PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
		})
		if err != nil {
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *TestkubeAPI) UpdateSecretHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to update secret '%s'", name)

		namespace := c.Query("namespace")
		if namespace == "" {
			namespace = s.Namespace
		}

		// Deserialize resource
		var input *testkube.SecretUpdate
		err := c.BodyParser(&input)
		if err != nil {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		// Get the secret details
		secret, err := s.Clientset.CoreV1().Secrets(namespace).Get(c.Context(), name, metav1.GetOptions{})
		if err != nil {
			if IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get secret: %s", errPrefix, err))
		}

		// Disallow when it is not controlled by Testkube
		if secret.Labels["createdBy"] != "testkube" {
			if s.enableSecretsEndpoint {
				return s.Error(c, http.StatusForbidden, fmt.Errorf("%s: secret is not controlled by Testkube", errPrefix))
			} else {
				// Make it the same as when it's actually not found, to avoid blind search
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
			}
		}

		if input.Labels != nil {
			labels := *input.Labels
			if labels == nil {
				labels = map[string]string{}
			}
			labels["createdBy"] = "testkube"
			delete(labels, "testkubeOwner")
			if secret.Labels["testkubeOwner"] != "" {
				labels["testkubeOwner"] = secret.Labels["testkubeOwner"]
			}
			secret.Labels = labels
		}

		if input.Data != nil && *input.Data != nil {
			secret.Data = nil
			secret.StringData = *input.Data
		}

		if input.Owner != nil {
			owner := secrets.MapSecretOwnerAPIToKube(input.Owner)
			if owner == "" {
				secret.Labels["testkubeOwner"] = owner
			} else {
				delete(secret.Labels, "testkubeOwner")
			}
		}

		// Update the secret
		secret, err = s.Clientset.CoreV1().Secrets(namespace).Update(c.Context(), secret, metav1.UpdateOptions{})
		if err != nil {
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		return c.JSON(secrets.MapSecretKubeToAPI(secret))
	}
}

func (s *TestkubeAPI) GetSecretHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get secret '%s'", name)

		namespace := c.Query("namespace")
		if namespace == "" {
			namespace = s.Namespace
		}

		// Get the secret details
		secret, err := s.Clientset.CoreV1().Secrets(namespace).Get(c.Context(), name, metav1.GetOptions{})
		if err != nil {
			if IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get secret: %s", errPrefix, err))
		}

		// Make it the same as when it's actually not found when disabled, to avoid blind search
		if secret.Labels["createdBy"] != "testkube" && !s.enableSecretsEndpoint {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
		}

		return c.JSON(secrets.MapSecretKubeToAPI(secret))
	}
}
