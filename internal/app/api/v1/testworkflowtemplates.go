package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

func (s *TestkubeAPI) ListTestWorkflowTemplatesHandler() fiber.Handler {
	errPrefix := "failed to list test workflow templates"
	return func(c *fiber.Ctx) (err error) {
		templates, err := s.getFilteredTestWorkflowTemplateList(c)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
		}
		err = SendResourceList(c, "TestWorkflowTemplate", testworkflowsv1.GroupVersion, testworkflows.MapTestWorkflowTemplateKubeToAPI, templates.Items...)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) GetTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test workflow template '%s'", name)
		template, err := s.TestWorkflowTemplatesClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		err = SendResource(c, "TestWorkflowTemplate", testworkflowsv1.GroupVersion, testworkflows.MapTemplateKubeToAPI, template)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) DeleteTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test workflow template '%s'", name)
		err := s.TestWorkflowTemplatesClient.Delete(name)
		s.Metrics.IncDeleteTestWorkflowTemplate(err)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *TestkubeAPI) CreateTestWorkflowTemplateHandler() fiber.Handler {
	errPrefix := "failed to create test workflow template"
	return func(c *fiber.Ctx) (err error) {
		// Deserialize resource
		obj := new(testworkflowsv1.TestWorkflowTemplate)
		if HasYAML(c) {
			err = common.DeserializeCRD(obj, c.Body())
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
		} else {
			var v *testkube.TestWorkflowTemplate
			err = c.BodyParser(&v)
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
			obj = testworkflows.MapTemplateAPIToKube(v)
		}

		// Validate resource
		if obj == nil || obj.Name == "" {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("name is required"))
		}
		obj.Namespace = s.Namespace

		// Get information about execution namespace
		// TODO: Considering that the TestWorkflow may override it - should it create in all execution namespaces?
		execNamespace := obj.Namespace
		if obj.Spec.Job != nil && obj.Spec.Job.Namespace != "" {
			execNamespace = obj.Spec.Job.Namespace
		}

		// Handle secrets auto-creation
		secrets := s.SecretManager.Batch("tw-", obj.Name)
		err = testworkflowresolver.ExtractCredentialsInTemplate(obj, secrets.Append)
		if err != nil {
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		// Create the resource
		obj, err = s.TestWorkflowTemplatesClient.Create(obj)
		if err != nil {
			s.Metrics.IncCreateTestWorkflowTemplate(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		err = s.SecretManager.InsertBatch(c.Context(), execNamespace, secrets, &metav1.OwnerReference{
			APIVersion: testworkflowsv1.GroupVersion.String(),
			Kind:       testworkflowsv1.ResourceTemplate,
			Name:       obj.Name,
			UID:        obj.UID,
		})
		s.Metrics.IncCreateTestWorkflowTemplate(err)
		if err != nil {
			_ = s.TestWorkflowTemplatesClient.Delete(obj.Name)
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}
		s.sendCreateWorkflowTemplateTelemetry(c.Context(), obj)

		err = SendResource(c, "TestWorkflowTemplate", testworkflowsv1.GroupVersion, testworkflows.MapTemplateKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) UpdateTestWorkflowTemplateHandler() fiber.Handler {
	errPrefix := "failed to update test workflow template"
	return func(c *fiber.Ctx) (err error) {
		name := c.Params("id")

		// Deserialize resource
		obj := new(testworkflowsv1.TestWorkflowTemplate)
		if HasYAML(c) {
			err = common.DeserializeCRD(obj, c.Body())
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
		} else {
			var v *testkube.TestWorkflowTemplate
			err = c.BodyParser(&v)
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
			obj = testworkflows.MapTemplateAPIToKube(v)
		}

		// Read existing resource
		template, err := s.TestWorkflowTemplatesClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		initial := template.DeepCopy()

		// Validate resource
		if obj == nil {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("body is required"))
		}
		obj.Namespace = template.Namespace
		obj.Name = template.Name
		obj.ResourceVersion = template.ResourceVersion

		// Get information about execution namespace
		// TODO: Considering that the TestWorkflow may override it - should it create in all execution namespaces?
		execNamespace := obj.Namespace
		if obj.Spec.Job != nil && obj.Spec.Job.Namespace != "" {
			execNamespace = obj.Spec.Job.Namespace
		}

		// Handle secrets auto-creation
		secrets := s.SecretManager.Batch("tw-", obj.Name)
		err = testworkflowresolver.ExtractCredentialsInTemplate(obj, secrets.Append)
		if err != nil {
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		// Update the resource
		obj, err = s.TestWorkflowTemplatesClient.Update(obj)
		if err != nil {
			s.Metrics.IncUpdateTestWorkflowTemplate(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		err = s.SecretManager.InsertBatch(c.Context(), execNamespace, secrets, &metav1.OwnerReference{
			APIVersion: testworkflowsv1.GroupVersion.String(),
			Kind:       testworkflowsv1.ResourceTemplate,
			Name:       obj.Name,
			UID:        obj.UID,
		})
		s.Metrics.IncUpdateTestWorkflowTemplate(err)
		if err != nil {
			_, err = s.TestWorkflowTemplatesClient.Update(initial)
			if err != nil {
				s.Log.Errorf("failed to recover previous TestWorkflowTemplate state: %v", err)
			}
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		err = SendResource(c, "TestWorkflowTemplate", testworkflowsv1.GroupVersion, testworkflows.MapTemplateKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) getFilteredTestWorkflowTemplateList(c *fiber.Ctx) (*testworkflowsv1.TestWorkflowTemplateList, error) {
	crTemplates, err := s.TestWorkflowTemplatesClient.List(c.Query("selector"))
	if err != nil {
		return nil, err
	}

	search := c.Query("textSearch")
	if search != "" {
		search = strings.ReplaceAll(search, "/", "--")
		for i := len(crTemplates.Items) - 1; i >= 0; i-- {
			if !strings.Contains(crTemplates.Items[i].Name, search) {
				crTemplates.Items = append(crTemplates.Items[:i], crTemplates.Items[i+1:]...)
			}
		}
	}

	return crTemplates, nil
}
