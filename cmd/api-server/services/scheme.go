package services

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	testbundlesv1 "github.com/kubeshop/testkube/api/testbundles/v1alpha1"
	testexecutionv1 "github.com/kubeshop/testkube/api/testexecution/v1"
	testsuiteexecutionv1 "github.com/kubeshop/testkube/api/testsuiteexecution/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

// NewControllerScheme returns the scheme of the Kubernetes controller manager: the Testkube
// resources it syncs and runs, and core/v1, whose ConfigMaps the Skill sync watches.
func NewControllerScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	for _, add := range []struct {
		what string
		add  func(*runtime.Scheme) error
	}{
		{"core/v1", corev1.AddToScheme},
		{"TestWorkflows", testworkflowsv1.AddToScheme},
		{"TestTriggers", testtriggersv1.AddToScheme},
		{"WorkflowTriggers", workflowtriggersv1.AddToScheme},
		{"Webhooks", executorv1.AddToScheme},
		{"Boards", insightsv1.AddToScheme},
		{"AI Agents, MCP servers, LLM providers and Skills", aiv1.AddToScheme},
		{"Test Bundles", testbundlesv1.AddToScheme},
		// Legacy schemes
		{"TestExecution", testexecutionv1.AddToScheme},
		{"TestSuiteExecution", testsuiteexecutionv1.AddToScheme},
	} {
		if err := add.add(scheme); err != nil {
			return nil, fmt.Errorf("add %s to kubernetes runtime scheme: %w", add.what, err)
		}
	}
	return scheme, nil
}
