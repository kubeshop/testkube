package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	templatesv1 "github.com/kubeshop/testkube/api/template/v1"
)

func TestTemplates(t *testing.T) {
	var tClient *TemplatesClient
	tType := templatesv1.CONTAINER_TemplateType
	testTemplates := []*templatesv1.Template{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-template1",
				Namespace: "test-ns",
			},
			Spec: templatesv1.TemplateSpec{
				Type_: &tType,
				Body:  "body1",
			},
			Status: templatesv1.TemplateStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-template2",
				Namespace: "test-ns",
			},
			Spec: templatesv1.TemplateSpec{
				Type_: &tType,
				Body:  "body2",
			},
			Status: templatesv1.TemplateStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-template3",
				Namespace: "test-ns",
			},
			Spec: templatesv1.TemplateSpec{
				Type_: &tType,
				Body:  "body3",
			},
			Status: templatesv1.TemplateStatus{},
		},
	}

	t.Run("NewTemplatesClient", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder()

		groupVersion := schema.GroupVersion{Group: "template.testkube.io", Version: "v1"}
		schemaBuilder := scheme.Builder{GroupVersion: groupVersion}
		schemaBuilder.Register(&templatesv1.TemplateList{})
		schemaBuilder.Register(&templatesv1.Template{})

		schema, err := schemaBuilder.Build()
		assert.NoError(t, err)
		assert.NotEmpty(t, schema)
		clientBuilder.WithScheme(schema)

		kClient := clientBuilder.Build()
		testNamespace := "test-ns"
		tClient = NewClient(kClient, testNamespace)
		assert.NotEmpty(t, tClient)
		assert.Equal(t, testNamespace, tClient.namespace)
	})
	t.Run("TemplateCreate", func(t *testing.T) {
		t.Run("Create new templates", func(t *testing.T) {
			for _, tp := range testTemplates {
				created, err := tClient.Create(tp)
				assert.NoError(t, err)
				assert.Equal(t, tp.Name, created.Name)

				res, err := tClient.Get(tp.Name)
				assert.NoError(t, err)
				assert.Equal(t, tp.Name, res.Name)
			}
		})
	})
	t.Run("TemplateUpdate", func(t *testing.T) {
		t.Run("Update new templates", func(t *testing.T) {
			for _, tp := range testTemplates {
				res, err := tClient.Get(tp.Name)
				assert.NoError(t, err)
				assert.Equal(t, tp.Name, res.Name)

				updated, err := tClient.Update(tp)
				assert.NoError(t, err)
				assert.Equal(t, tp.Name, updated.Name)
			}
		})
	})
	t.Run("TemplateList", func(t *testing.T) {
		t.Run("List without selector", func(t *testing.T) {
			l, err := tClient.List("")
			assert.NoError(t, err)
			assert.Equal(t, len(testTemplates), len(l.Items))
		})
	})
	t.Run("TemplateGet", func(t *testing.T) {
		t.Run("Get template with empty name", func(t *testing.T) {
			t.Parallel()
			_, err := tClient.Get("")
			assert.Error(t, err)
		})

		t.Run("Get template with non existent name", func(t *testing.T) {
			t.Parallel()
			_, err := tClient.Get("no-template")
			assert.Error(t, err)
		})

		t.Run("Get existing template", func(t *testing.T) {
			res, err := tClient.Get(testTemplates[0].Name)
			assert.NoError(t, err)
			assert.Equal(t, testTemplates[0].Name, res.Name)
		})
	})
	t.Run("TemplateDelete", func(t *testing.T) {
		t.Run("Delete items", func(t *testing.T) {
			for _, template := range testTemplates {
				tp, err := tClient.Get(template.Name)
				assert.NoError(t, err)
				assert.Equal(t, tp.Name, template.Name)

				err = tClient.Delete(template.Name)
				assert.NoError(t, err)

				_, err = tClient.Get(template.Name)
				assert.Error(t, err)
			}
		})

		t.Run("Delete non-existent item", func(t *testing.T) {
			_, err := tClient.Get("no-template")
			assert.Error(t, err)

			err = tClient.Delete("no-template")
			assert.Error(t, err)
		})
	})
}
