//go:build k8sIntegration

// TODO set-up workflows which can run kubernetes related tests

package v3

import (
	"testing"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsuitev3 "github.com/kubeshop/testkube/api/testsuite/v3"
	kubeclient "github.com/kubeshop/testkube/pkg/operator/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClient_IntegrationWithSecrets(t *testing.T) {
	const testsuiteName = "testsuite-example-with-secrets"
	// given test client and example test
	client, err := kubeclient.GetClient()
	assert.NoError(t, err)

	c := NewClient(client, "testkube")

	tst0, err := c.Create(&testsuitev3.TestSuite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testsuiteName,
			Namespace: "testkube",
		},
		Spec: testsuitev3.TestSuiteSpec{
			ExecutionRequest: &testsuitev3.TestSuiteExecutionRequest{
				Variables: map[string]testsuitev3.Variable{
					"secretVar1": {
						Type_: commonv1.VariableTypeSecret,
						Name:  "secretVar1",
						Value: "SECR3t",
					},
					"secretVar2": {
						Type_: commonv1.VariableTypeSecret,
						Name:  "secretVar2",
						Value: "SomeOtherSecretVar",
					},
				},
			},
		},
	})

	assert.NoError(t, err)

	// when update test secret variable
	secret := tst0.Spec.ExecutionRequest.Variables["secretVar1"]
	secret.Value = "UpdatedSecretValue"
	tst0.Spec.ExecutionRequest.Variables["secretVar1"] = secret

	secret = tst0.Spec.ExecutionRequest.Variables["secretVar2"]
	secret.Value = "SomeOtherSecretVar"
	tst0.Spec.ExecutionRequest.Variables["secretVar2"] = secret

	tstUpdated, err := c.Update(tst0)
	assert.NoError(t, err)

	// then value should be updated
	tst1, err := c.Get(tst0.Name)
	assert.NoError(t, err)

	assert.Equal(t, "UpdatedSecretValue", tst1.Spec.ExecutionRequest.Variables["secretVar1"].Value)
	assert.Equal(t, "SomeOtherSecretVar", tst1.Spec.ExecutionRequest.Variables["secretVar2"].Value)

	// when test is deleted
	err = c.Delete(tstUpdated.Name)
	assert.NoError(t, err)

	// then there should be no test anymore
	tst2, err := c.Get(tst0.Name)
	assert.Nil(t, tst2)
	assert.Error(t, err)

}

func TestClient_IntegrationWithoutSecrets(t *testing.T) {
	const testsuiteName = "testsuite-example-without-secrets"
	// given test client and example test
	client, err := kubeclient.GetClient()
	assert.NoError(t, err)

	c := NewClient(client, "testkube")

	tst0, err := c.Create(&testsuitev3.TestSuite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testsuiteName,
			Namespace: "testkube",
		},
		Spec: testsuitev3.TestSuiteSpec{
			ExecutionRequest: &testsuitev3.TestSuiteExecutionRequest{
				Variables: map[string]testsuitev3.Variable{
					"secretVar1": {
						Type_: commonv1.VariableTypeBasic,
						Name:  "var1",
						Value: "val1",
					},
				},
			},
		},
	})

	assert.NoError(t, err)

	// when update test secret variable
	secret := tst0.Spec.ExecutionRequest.Variables["secretVar1"]
	secret.Value = "updatedval"
	tst0.Spec.ExecutionRequest.Variables["var1"] = secret

	tstUpdated, err := c.Update(tst0)
	assert.NoError(t, err)

	// then value should be updated
	tst1, err := c.Get(tst0.Name)
	assert.NoError(t, err)

	assert.Equal(t, "updatedval", tst1.Spec.ExecutionRequest.Variables["var1"].Value)

	// when test is deleted
	err = c.Delete(tstUpdated.Name)
	assert.NoError(t, err)

	// then there should be no test anymore
	tst2, err := c.Get(tst0.Name)
	assert.Nil(t, tst2)
	assert.Error(t, err)

}
