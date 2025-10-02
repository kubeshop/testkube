package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/secretmanager"
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

		results, err := s.SecretManager.List(c.Context(), namespace, all)
		if errors.Is(err, secretmanager.ErrManagementDisabled) {
			return s.Error(c, http.StatusForbidden, errors.Wrap(err, errPrefix))
		} else if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list secrets: %s", errPrefix, err))
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

		// Load the owner
		var owner *metav1.OwnerReference
		if input.Owner != nil && input.Owner.Kind != "" && input.Owner.Name != "" {
			ownerRef, err := s.fetchOwnerReference(input.Owner.Kind, input.Owner.Name)
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid owner", err)
			}
			owner = &ownerRef
		}

		// Create the resource
		secret, err := s.SecretManager.Create(c.Context(), input.Namespace, input.Name, input.Data, secretmanager.CreateOptions{
			Labels: input.Labels,
			Owner:  owner,
		})
		if errors.Is(err, secretmanager.ErrCreateDisabled) {
			return s.Error(c, http.StatusForbidden, errors.Wrap(err, errPrefix))
		} else if err != nil {
			return s.Error(c, http.StatusBadGateway, errors.Wrap(err, errPrefix))
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

		err := s.SecretManager.Delete(c.Context(), namespace, name)
		if errors.Is(err, secretmanager.ErrDeleteDisabled) {
			return s.Error(c, http.StatusForbidden, errors.Wrap(err, errPrefix))
		} else if apiutils.IsNotFound(err) {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
		} else if err != nil {
			return s.Error(c, http.StatusBadGateway, errors.Wrap(err, errPrefix))
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

		// Load the owner
		var owner *metav1.OwnerReference
		if input.Owner != nil {
			if input.Owner.Kind != "" && input.Owner.Name != "" {
				ownerRef, err := s.fetchOwnerReference(input.Owner.Kind, input.Owner.Name)
				if err != nil {
					return s.BadRequest(c, errPrefix, "invalid owner", err)
				}
				owner = &ownerRef
			} else {
				owner = &metav1.OwnerReference{}
			}
		}

		var data map[string]string
		if input.Data != nil && *input.Data != nil {
			data = *input.Data
		}

		var labels map[string]string
		if input.Labels != nil && *input.Labels != nil {
			labels = *input.Labels
		}

		secret, err := s.SecretManager.Update(c.Context(), namespace, name, data, secretmanager.UpdateOptions{
			Labels: labels,
			Owner:  owner,
		})
		if errors.Is(err, secretmanager.ErrModifyDisabled) {
			return s.Error(c, http.StatusForbidden, errors.Wrap(err, errPrefix))
		} else if apiutils.IsNotFound(err) {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
		} else if err != nil {
			return s.Error(c, http.StatusBadGateway, errors.Wrap(err, errPrefix))
		}

		return c.JSON(secret)
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
		secret, err := s.SecretManager.Get(c.Context(), namespace, name)
		if apiutils.IsNotFound(err) {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: secret not found", errPrefix))
		} else if errors.Is(err, secretmanager.ErrManagementDisabled) {
			return s.Error(c, http.StatusForbidden, errors.Wrap(err, errPrefix))
		} else if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get secret: %s", errPrefix, err))
		}
		return c.JSON(secret)
	}
}

func (s *TestkubeAPI) fetchOwnerReference(kind, name string) (metav1.OwnerReference, error) {
	if kind == testworkflowsv1.Resource {
		obj, err := s.TestWorkflowsK8SClient.Get(name)
		if err != nil {
			return metav1.OwnerReference{}, errors.Wrap(err, "fetching owner")
		}

		// Use AppendTypeMeta to set the GroupVersionKind properly
		crdcommon.AppendTypeMeta("TestWorkflow", testworkflowsv1.GroupVersion, obj)

		ownerRef := metav1.OwnerReference{
			APIVersion: obj.APIVersion,
			Kind:       obj.Kind,
			Name:       obj.Name,
			UID:        obj.UID,
		}
		return ownerRef, nil
	} else if kind == testworkflowsv1.ResourceTemplate {
		obj, err := s.TestWorkflowTemplatesK8SClient.Get(name)
		if err != nil {
			return metav1.OwnerReference{}, errors.Wrap(err, "fetching owner")
		}

		// Use AppendTypeMeta to set the GroupVersionKind properly
		crdcommon.AppendTypeMeta("TestWorkflowTemplate", testworkflowsv1.GroupVersion, obj)

		ownerRef := metav1.OwnerReference{
			APIVersion: obj.APIVersion,
			Kind:       obj.Kind,
			Name:       obj.Name,
			UID:        obj.UID,
		}
		return ownerRef, nil
	}

	return metav1.OwnerReference{}, fmt.Errorf("unsupported owner kind: %s", kind)
}
