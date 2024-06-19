package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
			// Fetch the available keys
			keys := make([]string, 0, len(secret.Data)+len(secret.StringData))
			for k := range secret.Data {
				keys = append(keys, k)
			}
			for k := range secret.StringData {
				keys = append(keys, k)
			}

			// Fetch ownership details
			var owner *testkube.SecretOwner
			kind, name, _ := strings.Cut(secret.Labels["testkubeOwner"], "/")
			if kind != "" && name != "" {
				owner = &testkube.SecretOwner{
					Kind: common.Ptr(testkube.SecretOwnerKind(kind)),
					Name: name,
				}
			}

			// Ensure it's not created externally
			controlled := secret.Labels["createdBy"] == "testkube"

			// Clean up the labels
			delete(secret.Labels, "createdBy")
			delete(secret.Labels, "testkubeOwner")
			if len(secret.Labels) == 0 {
				secret.Labels = nil
			}

			results[i] = testkube.Secret{
				Name:       secret.Name,
				Labels:     secret.Labels,
				Controlled: controlled,
				Owner:      owner,
				Keys:       keys,
			}
		}

		return c.JSON(results)
	}
}

func (s *TestkubeAPI) CreateSecretHandler() fiber.Handler {
	errPrefix := "failed to create secret"
	return func(c *fiber.Ctx) (err error) {
		// Deserialize resource
		var v *testkube.SecretInput
		err = c.BodyParser(&v)
		if err != nil {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		// Validate resource
		if v == nil || v.Name == "" {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("name is required"))
		}
		if len(v.Data) == 0 {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("data should not be empty"))
		}

		// Apply defaults
		if v.Namespace == "" {
			v.Namespace = s.Namespace
		}
		if v.Labels == nil {
			v.Labels = map[string]string{}
		}
		v.Labels["createdBy"] = "testkube"
		if v.Owner != nil && v.Owner.Kind != nil && *v.Owner.Kind != "" && v.Owner.Name != "" {
			v.Labels["testkubeOwner"] = fmt.Sprintf("%s/%s", v.Owner.Kind, v.Owner.Name)
		} else {
			delete(v.Labels, "testkubeOwner")
		}

		// Create the resource
		err = s.SecretClient.Create(v.Name, v.Labels, v.Data, v.Namespace)
		if err != nil {
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		c.Status(http.StatusNoContent)
		return
	}
}
