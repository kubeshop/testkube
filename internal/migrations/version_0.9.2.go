package migrations

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube-operator/api/tests/v1"
	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsuite "github.com/kubeshop/testkube-operator/api/testsuite/v2"
	scriptsclientv2 "github.com/kubeshop/testkube-operator/pkg/client/scripts/v2"
	testsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/tests"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/migrator"
)

func NewVersion_0_9_2(
	scriptsClient *scriptsclientv2.ScriptsClient,
	testsClientV1 *testsclientv1.TestsClient,
	testsClientV3 *testsclientv3.TestsClient,
	testsuitesClient *testsuitesclientv2.TestSuitesClient,
) *Version_0_9_2 {
	return &Version_0_9_2{
		scriptsClient:    scriptsClient,
		testsClientV1:    testsClientV1,
		testsClientV3:    testsClientV3,
		testsuitesClient: testsuitesClient,
	}
}

type Version_0_9_2 struct {
	scriptsClient    *scriptsclientv2.ScriptsClient
	testsClientV1    *testsclientv1.TestsClient
	testsClientV3    *testsclientv3.TestsClient
	testsuitesClient *testsuitesclientv2.TestSuitesClient
}

func (m *Version_0_9_2) Version() string {
	return "0.9.2"
}
func (m *Version_0_9_2) Migrate() error {
	scripts, err := m.scriptsClient.List(nil)
	if err != nil {
		return err
	}

	for _, script := range scripts.Items {
		if _, err = m.testsClientV3.Get(script.Name); err != nil && !errors.IsNotFound(err) {
			return err
		}

		if err == nil {
			continue
		}

		test := &testsv3.Test{
			ObjectMeta: metav1.ObjectMeta{
				Name:      script.Name,
				Namespace: script.Namespace,
			},
			Spec: testsv3.TestSpec{
				Type_: script.Spec.Type_,
				Name:  script.Spec.Name,
			},
		}

		if len(script.Spec.Params) != 0 {
			test.Spec.ExecutionRequest = &testsv3.ExecutionRequest{
				Variables: make(map[string]testsv3.Variable, len(script.Spec.Params)),
			}

			for key, value := range script.Spec.Params {
				test.Spec.ExecutionRequest.Variables[key] = testsv3.Variable{
					Name:  key,
					Value: value,
					Type_: string(*testkube.VariableTypeBasic),
				}
			}
		}

		if script.Spec.Content != nil {
			test.Spec.Content = &testsv3.TestContent{
				Type_: testsv3.TestContentType(script.Spec.Content.Type_),
				Data:  script.Spec.Content.Data,
				Uri:   script.Spec.Content.Uri,
			}

			if script.Spec.Content.Repository != nil {
				test.Spec.Content.Repository = &testsv3.Repository{
					Type_:  script.Spec.Content.Repository.Type_,
					Uri:    script.Spec.Content.Repository.Uri,
					Branch: script.Spec.Content.Repository.Branch,
					Path:   script.Spec.Content.Repository.Path,
				}
			}
		}

		if _, err = m.testsClientV3.Create(test, false); err != nil {
			return err
		}

		if err = m.scriptsClient.Delete(script.Name); err != nil {
			return err
		}
	}

	tests, err := m.testsClientV1.List(nil)
	if err != nil {
		return err
	}

OUTER:
	for _, test := range tests.Items {
		if _, err = m.testsuitesClient.Get(test.Name); err != nil && !errors.IsNotFound(err) {
			return err
		}

		if err == nil {
			continue
		}

		for _, managedField := range test.GetManagedFields() {
			if !strings.HasSuffix(managedField.APIVersion, "/v1") {
				continue OUTER
			}
		}

		testsuite := &testsuite.TestSuite{
			ObjectMeta: metav1.ObjectMeta{
				Name:      test.Name,
				Namespace: test.Namespace,
			},
			Spec: testsuite.TestSuiteSpec{
				Repeats:     test.Spec.Repeats,
				Description: test.Spec.Description,
			},
		}

		for _, step := range test.Spec.Before {
			testsuite.Spec.Before = append(testsuite.Spec.Before, copyTestStepTest2Testsuite(step))
		}

		for _, step := range test.Spec.Steps {
			testsuite.Spec.Steps = append(testsuite.Spec.Steps, copyTestStepTest2Testsuite(step))
		}

		for _, step := range test.Spec.After {
			testsuite.Spec.After = append(testsuite.Spec.After, copyTestStepTest2Testsuite(step))
		}

		if _, err = m.testsuitesClient.Create(testsuite); err != nil {
			return err
		}

		if err = m.testsClientV1.Delete(test.Name); err != nil {
			return err
		}
	}

	return nil
}
func (m *Version_0_9_2) Info() string {
	return "Moving scripts v2 resources to tests v2 ones and tests v1 resources to testsuites v1 ones"
}

func (m *Version_0_9_2) Type() migrator.MigrationType {
	return migrator.MigrationTypeServer
}

func copyTestStepTest2Testsuite(step testsv1.TestStepSpec) testsuite.TestSuiteStepSpec {
	result := testsuite.TestSuiteStepSpec{
		Type: testsuite.TestSuiteStepType(step.Type),
	}

	if step.Execute != nil {
		result.Execute = &testsuite.TestSuiteStepExecute{
			Namespace:     step.Execute.Namespace,
			Name:          step.Execute.Name,
			StopOnFailure: step.Execute.StopOnFailure,
		}
	}

	if step.Delay != nil {
		result.Delay = &testsuite.TestSuiteStepDelay{
			Duration: step.Delay.Duration,
		}
	}

	return result
}
