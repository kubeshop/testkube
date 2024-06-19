package v1

import (
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
func (s TestkubeAPI) ListSecretsHandler() fiber.Handler {
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
			keys := make([]string, 0, len(secret.Data)+len(secret.StringData))
			for k := range secret.Data {
				keys = append(keys, k)
			}
			for k := range secret.StringData {
				keys = append(keys, k)
			}
			var owner *testkube.SecretOwner
			kind, name, _ := strings.Cut(secret.Labels["testkubeOwner"], "/")
			if kind != "" && name != "" {
				owner = &testkube.SecretOwner{
					Kind: common.Ptr(testkube.SecretOwnerKind(kind)),
					Name: name,
				}
			}
			results[i] = testkube.Secret{
				Name:       secret.Name,
				Controlled: secret.Labels["createdBy"] == "testkube",
				Owner:      owner,
				Keys:       keys,
			}
		}

		return c.JSON(results)
	}
}
