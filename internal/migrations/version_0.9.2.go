package migrations

import (
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube-operator/apis/tests/v1"
	testsv2 "github.com/kubeshop/testkube-operator/apis/tests/v2"
	testsuite "github.com/kubeshop/testkube-operator/apis/testsuite/v1"
	scriptsclientv2 "github.com/kubeshop/testkube-operator/client/scripts/v2"
	testsclientv1 "github.com/kubeshop/testkube-operator/client/tests"
	testsclientv2 "github.com/kubeshop/testkube-operator/client/tests/v2"
	testsuitesclientv1 "github.com/kubeshop/testkube-operator/client/testsuites/v1"
)

func NewVersion_0_9_2(
	scriptsClient *scriptsclientv2.ScriptsClient,
	testsClientV1 *testsclientv1.TestsClient,
	testsClientV2 *testsclientv2.TestsClient,
	testsuitesClient *testsuitesclientv1.TestSuitesClient,
) *Version_0_9_2 {
	return &Version_0_9_2{
		scriptsClient:    scriptsClient,
		testsClientV1:    testsClientV1,
		testsClientV2:    testsClientV2,
		testsuitesClient: testsuitesClient,
	}
}

type Version_0_9_2 struct {
	scriptsClient    *scriptsclientv2.ScriptsClient
	testsClientV1    *testsclientv1.TestsClient
	testsClientV2    *testsclientv2.TestsClient
	testsuitesClient *testsuitesclientv1.TestSuitesClient
	namespace        string
}

func (m *Version_0_9_2) Version() string {
	return "0.9.2"
}
func (m *Version_0_9_2) Migrate() error {
	namespace := os.Getenv("TESTKUBE_NAMESPACE")

	scripts, err := m.scriptsClient.List(namespace, nil)
	if err != nil {
		return err
	}

	for _, script := range scripts.Items {
		if _, err = m.testsClientV2.Get(namespace, script.Name); err != nil && !errors.IsNotFound(err) {
			return err
		}

		if err == nil {
			continue
		}

		test := &testsv2.Test{
			ObjectMeta: metav1.ObjectMeta{
				Name:      script.Name,
				Namespace: script.Namespace,
			},
			Spec: testsv2.TestSpec{
				Type_:  script.Spec.Type_,
				Name:   script.Spec.Name,
				Params: script.Spec.Params,
				Tags:   script.Spec.Tags,
			},
		}

		if script.Spec.Content != nil {
			test.Spec.Content = &testsv2.TestContent{
				Type_: script.Spec.Content.Type_,
				Data:  script.Spec.Content.Data,
				Uri:   script.Spec.Content.Uri,
			}

			if script.Spec.Content.Repository != nil {
				test.Spec.Content.Repository = &testsv2.Repository{
					Type_:  script.Spec.Content.Repository.Type_,
					Uri:    script.Spec.Content.Repository.Uri,
					Branch: script.Spec.Content.Repository.Branch,
					Path:   script.Spec.Content.Repository.Path,
				}
			}
		}

		if _, err = m.testsClientV2.Create(test); err != nil {
			return err
		}
	}

	if err = m.scriptsClient.DeleteAll(namespace); err != nil {
		return err
	}

	tests, err := m.testsClientV1.List(namespace, nil)
	if err != nil {
		return err
	}

	for _, test := range tests.Items {
		if _, err = m.testsuitesClient.Get(namespace, test.Name); err != nil && !errors.IsNotFound(err) {
			return err
		}

		if err == nil {
			continue
		}

		testsuite := &testsuite.TestSuite{
			ObjectMeta: metav1.ObjectMeta{
				Name:      test.Name,
				Namespace: test.Namespace,
			},
			Spec: testsuite.TestSuiteSpec{
				Repeats:     test.Spec.Repeats,
				Description: test.Spec.Description,
				Tags:        test.Spec.Tags,
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
	}

	if err = m.testsClientV1.DeleteAll(namespace); err != nil {
		return err
	}

	return nil
}
func (m *Version_0_9_2) Info() string {
	return "Moving scripts v1 resources to tests v2 ones and tests v1 resources to testsuites v1 ones"
}

func (m *Version_0_9_2) IsClient() bool {
	return false
}

func copyTestStepTest2Testsuite(step testsv1.TestStepSpec) testsuite.TestSuiteStepSpec {
	result := testsuite.TestSuiteStepSpec{
		Type: step.Type,
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
