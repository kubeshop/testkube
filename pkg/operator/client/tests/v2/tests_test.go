//go:build k8sIntegration

// TODO set-up workflows which can run kubernetes related tests

package tests

import (
	"fmt"
	"testing"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsv2 "github.com/kubeshop/testkube/api/tests/v2"
	kubeclient "github.com/kubeshop/testkube/pkg/operator/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClient_IntegrationWithSecrets(t *testing.T) {
	// given test client and example test
	client, err := kubeclient.GetClient()
	assert.NoError(t, err)

	c := NewClient(client, "testkube")
	testSpec := &testsv2.Test{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-example-with-secrets",
			Namespace: "testkube",
		},
		Spec: testsv2.TestSpec{
			Type_: "postman/collection",
			Content: &testsv2.TestContent{
				Data: "{}",
			},
			Variables: map[string]testsv2.Variable{
				"var1": {
					Type_: commonv1.VariableTypeSecret,
					Name:  "var1",
					Value: "val1",
				},
				"var2": {
					Type_: commonv1.VariableTypeSecret,
					Name:  "var2",
					Value: "val2",
				},
			},
		},
	}

	// when create test
	test1, err := c.Create(testSpec)
	assert.NoError(t, err)

	// then value should be updated
	test2, err := c.Get(test1.Name)
	assert.NoError(t, err)
	assert.Equal(t, "val1", test2.Spec.Variables["var1"].Value)
	assert.Equal(t, "val2", test2.Spec.Variables["var2"].Value)

	// when and update test secret variable
	secret := test2.Spec.Variables["var1"]
	secret.Value = "updated1"
	test2.Spec.Variables["var1"] = secret

	test3, err := c.Update(test2)
	assert.NoError(t, err)

	// then value should be updated
	test4, err := c.Get(test3.Name)
	assert.NoError(t, err)
	assert.Equal(t, "updated1", test4.Spec.Variables["var1"].Value)
	assert.Equal(t, "val2", test4.Spec.Variables["var2"].Value)

	// when test is deleted
	err = c.Delete(test4.Name)
	assert.NoError(t, err)

	// then there should be no test anymore
	test5, err := c.Get(test4.Name)
	assert.Nil(t, test5)
	assert.Error(t, err)
}

func TestClient_IntegrationWithoutSecrets(t *testing.T) {
	// given test client and example test
	client, err := kubeclient.GetClient()
	assert.NoError(t, err)

	c := NewClient(client, "testkube")
	testSpec := &testsv2.Test{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-example-without-secrets",
			Namespace: "testkube",
		},
		Spec: testsv2.TestSpec{
			Type_: "postman/collection",
			Content: &testsv2.TestContent{
				Data: "{}",
			},
			Variables: map[string]testsv2.Variable{
				"var1": {
					Type_: commonv1.VariableTypeBasic,
					Name:  "var1",
					Value: "val1",
				},
			},
		},
	}

	// when create test
	test1, err := c.Create(testSpec)
	assert.NoError(t, err)

	// then value should be updated
	test2, err := c.Get(test1.Name)

	assert.NoError(t, err)
	assert.Equal(t, "val1", test2.Spec.Variables["var1"].Value)

	// when and update test variable variable
	variable := test2.Spec.Variables["var1"]
	variable.Value = "updated1"
	test2.Spec.Variables["var1"] = variable

	test3, err := c.Update(test2)
	assert.NoError(t, err)

	// then value should be updated
	test4, err := c.Get(test3.Name)
	assert.NoError(t, err)
	assert.Equal(t, "updated1", test4.Spec.Variables["var1"].Value)

	fmt.Printf("%+v\n", test4.Name)

	// when test is deleted
	err = c.Delete(test4.Name)
	assert.NoError(t, err)

	// then there should be no test anymore
	test5, err := c.Get(test4.Name)
	assert.Nil(t, test5)
	assert.Error(t, err)
}
