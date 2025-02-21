package common

import (
	"fmt"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	internalcommon "github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	testsuitesmapper "github.com/kubeshop/testkube/pkg/mapper/testsuites"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
)

// OfficialTestWorkflowTemplates contains official test workflow templates
var OfficialTestWorkflowTemplates = map[string]struct {
	Name           string
	ConfigRun      string
	NeedWorkingDir bool
	NeedTestFile   bool
}{
	"cypress-executor":    {Name: "official--cypress--beta", ConfigRun: "npx cypress run", NeedWorkingDir: true},
	"k6-executor":         {Name: "official--k6--beta", ConfigRun: "k6 run", NeedTestFile: true},
	"playwright-executor": {Name: "official--playwright--beta", ConfigRun: "npx playwright test", NeedWorkingDir: true},
	"postman-executor":    {Name: "official--postman--beta", ConfigRun: "newman run", NeedTestFile: true},
}

// UIPrintCRD prints crd to ui
func UIPrintCRD(tmpl crd.Template, object any, firstEntry *bool) {
	data, err := crd.ExecuteTemplate(tmpl, object)
	ui.ExitOnError("executing crd template", err)
	if !*firstEntry {
		fmt.Printf("\n---\n")
	} else {
		*firstEntry = false
	}
	fmt.Print(data)
}

// PrintTestWorkflowCRDForTest prints test workflow CRD for Test
func PrintTestWorkflowCRDForTest(test testkube.Test, templateName string, options testworkflowmappers.Options) {
	testCR := testsmapper.MapTestAPIToCR(test)
	testWorkflow := testworkflowmappers.MapTestKubeToTestWorkflowKube(testCR, templateName, options)
	b, err := internalcommon.SerializeCRDs([]testworkflowsv1.TestWorkflow{testWorkflow}, internalcommon.SerializeOptions{
		OmitCreationTimestamp: true,
		CleanMeta:             true,
		Kind:                  testworkflowsv1.Resource,
		GroupVersion:          &testworkflowsv1.GroupVersion,
	})
	ui.ExitOnError("serializing obj", err)
	fmt.Print(string(b))
}

// PrintTestWorkflowCRDForTestSuite prints test workflow CRD for Test Suite
func PrintTestWorkflowCRDForTestSuite(testSuite testkube.TestSuite) {
	testSuiteCR, err := testsuitesmapper.MapAPIToCR(testSuite)
	ui.ExitOnError("mapping obj", err)

	testWorkflow := testworkflowmappers.MapTestSuiteKubeToTestWorkflowKube(testSuiteCR)
	b, err := internalcommon.SerializeCRDs([]testworkflowsv1.TestWorkflow{testWorkflow}, internalcommon.SerializeOptions{
		OmitCreationTimestamp: true,
		CleanMeta:             true,
		Kind:                  testworkflowsv1.Resource,
		GroupVersion:          &testworkflowsv1.GroupVersion,
	})
	ui.ExitOnError("serializing obj", err)
	fmt.Print(string(b))
}

// PrintTestWorkflowTemplateCRDForExecutor prints test workflow template CRD for Executor
func PrintTestWorkflowTemplateCRDForExecutor(executor testkube.ExecutorDetails, namespace string) {
	executorCR := executorsmapper.MapExecutorDetailsToExecutorCRD(executor, namespace)
	testWorkflowTemplate := testworkflowmappers.MapExecutorKubeToTestWorkflowTemplateKube(executorCR)
	b, err := internalcommon.SerializeCRDs([]testworkflowsv1.TestWorkflowTemplate{testWorkflowTemplate}, internalcommon.SerializeOptions{
		OmitCreationTimestamp: true,
		CleanMeta:             true,
		Kind:                  testworkflowsv1.ResourceTemplate,
		GroupVersion:          &testworkflowsv1.GroupVersion,
	})
	ui.ExitOnError("serializing obj", err)
	fmt.Print(string(b))
}
