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
	"github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

func (s *TestkubeAPI) ListTestWorkflowTemplatesHandler() fiber.Handler {
	errPrefix := "failed to list test workflow templates"
	return func(c *fiber.Ctx) (err error) {
		templates, err := s.getFilteredTestWorkflowTemplateList(c)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
		}
		crTemplates := common.MapSlice(templates, func(w testkube.TestWorkflowTemplate) testworkflowsv1.TestWorkflowTemplate {
			return *testworkflows.MapTemplateAPIToKube(&w)
		})
		err = SendResourceList(c, "TestWorkflowTemplate", testworkflowsv1.GroupVersion, testworkflows.MapTestWorkflowTemplateKubeToAPI, crTemplates...)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) GetTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test workflow template '%s'", name)
		template, err := s.TestWorkflowTemplatesClient.Get(ctx, environmentId, name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		crTemplate := testworkflows.MapTemplateAPIToKube(template)
		err = SendResource(c, "TestWorkflowTemplate", testworkflowsv1.GroupVersion, testworkflows.MapTemplateKubeToAPI, crTemplate)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) DeleteTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test workflow template '%s'", name)
		err := s.TestWorkflowTemplatesClient.Delete(ctx, environmentId, name)
		s.Metrics.IncDeleteTestWorkflowTemplate(err)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *TestkubeAPI) DeleteTestWorkflowTemplatesHandler() fiber.Handler {
	errPrefix := "failed to delete test workflow templates"
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		selector := c.Query("selector")
		labelSelector, err := metav1.ParseToLabelSelector(selector)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		if len(labelSelector.MatchExpressions) > 0 {
			return s.ClientError(c, errPrefix, errors.New("matchExpressions are not supported"))
		}

		_, err = s.TestWorkflowTemplatesClient.DeleteByLabels(ctx, environmentId, labelSelector.MatchLabels)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *TestkubeAPI) CreateTestWorkflowTemplateHandler() fiber.Handler {
	errPrefix := "failed to create test workflow template"
	return func(c *fiber.Ctx) (err error) {
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		// Deserialize resource
		obj := new(testworkflowsv1.TestWorkflowTemplate)
		if HasYAML(c) {
			err = crdcommon.DeserializeCRD(obj, c.Body())
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
		err = s.TestWorkflowTemplatesClient.Create(ctx, environmentId, *testworkflows.MapTemplateKubeToAPI(obj))
		if err != nil {
			s.Metrics.IncCreateTestWorkflowTemplate(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		if secrets.HasData() {
			uid, _ := s.TestWorkflowTemplatesClient.GetKubernetesObjectUID(ctx, environmentId, obj.Name)
			var ref *metav1.OwnerReference
			if uid != "" {
				ref = &metav1.OwnerReference{
					APIVersion: testworkflowsv1.GroupVersion.String(),
					Kind:       testworkflowsv1.ResourceTemplate,
					Name:       obj.Name,
					UID:        uid,
				}
			}
			err = s.SecretManager.InsertBatch(c.Context(), execNamespace, secrets, ref)
			if err != nil {
				_ = s.TestWorkflowTemplatesClient.Delete(ctx, environmentId, obj.Name)
				return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
			}
		}
		s.Metrics.IncCreateTestWorkflowTemplate(err)
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
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		name := c.Params("id")

		// Deserialize resource
		obj := new(testworkflowsv1.TestWorkflowTemplate)
		if HasYAML(c) {
			err = crdcommon.DeserializeCRD(obj, c.Body())
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
		template, err := s.TestWorkflowTemplatesClient.Get(ctx, environmentId, name)
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
		err = s.TestWorkflowTemplatesClient.Update(ctx, environmentId, *testworkflows.MapTemplateKubeToAPI(obj))
		if err != nil {
			s.Metrics.IncUpdateTestWorkflowTemplate(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		if secrets.HasData() {
			uid, _ := s.TestWorkflowTemplatesClient.GetKubernetesObjectUID(ctx, environmentId, obj.Name)
			var ref *metav1.OwnerReference
			if uid != "" {
				ref = &metav1.OwnerReference{
					APIVersion: testworkflowsv1.GroupVersion.String(),
					Kind:       testworkflowsv1.ResourceTemplate,
					Name:       obj.Name,
					UID:        uid,
				}
			}
			err = s.SecretManager.InsertBatch(c.Context(), execNamespace, secrets, ref)
		}
		s.Metrics.IncUpdateTestWorkflowTemplate(err)
		if err != nil {
			err = s.TestWorkflowTemplatesClient.Update(ctx, environmentId, *initial)
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

func (s *TestkubeAPI) getFilteredTestWorkflowTemplateList(c *fiber.Ctx) ([]testkube.TestWorkflowTemplate, error) {
	ctx := c.Context()
	environmentId := s.getEnvironmentId()
	selector := c.Query("selector")
	labelSelector, err := metav1.ParseToLabelSelector(selector)
	if err != nil {
		return nil, err
	}
	if len(labelSelector.MatchExpressions) > 0 {
		return nil, errors.New("MatchExpressions are not supported")
	}

	templates, err := s.TestWorkflowTemplatesClient.List(ctx, environmentId, testworkflowtemplateclient.ListOptions{
		Labels: labelSelector.MatchLabels,
	})
	if err != nil {
		return nil, err
	}

	search := c.Query("textSearch")
	if search != "" {
		search = strings.ReplaceAll(search, "/", "--")
		for i := len(templates) - 1; i >= 0; i-- {
			if !strings.Contains(templates[i].Name, search) {
				templates = append(templates[:i], templates[i+1:]...)
			}
		}
	}

	return templates, nil
}
