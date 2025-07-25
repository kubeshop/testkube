/*
 * Testkube API
 *
 * Testkube provides a Kubernetes-native framework for test definition, execution and results
 *
 * API version: 1.0.0
 * Contact: contact@testkube.io
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package testkube

type TestWorkflowStep struct {
	// readable name for the step
	Name string `json:"name,omitempty"`
	// expression to declare under which conditions the step should be run; defaults to \"passed\", except artifacts where it defaults to \"always\"
	Condition string        `json:"condition,omitempty"`
	Pure      *BoxedBoolean `json:"pure,omitempty"`
	// should the step be paused initially
	Paused bool `json:"paused,omitempty"`
	// is the step expected to fail
	Negative bool `json:"negative,omitempty"`
	// is the step optional, so the failure won't affect the TestWorkflow result
	Optional bool `json:"optional,omitempty"`
	// list of TestWorkflowTemplates to use
	Use      []TestWorkflowTemplateRef `json:"use,omitempty"`
	Template *TestWorkflowTemplateRef  `json:"template,omitempty"`
	Retry    *TestWorkflowRetryPolicy  `json:"retry,omitempty"`
	// maximum time this step may take
	Timeout string `json:"timeout,omitempty"`
	// delay before the step
	Delay    string                             `json:"delay,omitempty"`
	Content  *TestWorkflowContent               `json:"content,omitempty"`
	Services map[string]TestWorkflowServiceSpec `json:"services,omitempty"`
	// script to run in a default shell for the container
	Shell      string                       `json:"shell,omitempty"`
	Run        *TestWorkflowStepRun         `json:"run,omitempty"`
	WorkingDir *BoxedString                 `json:"workingDir,omitempty"`
	Container  *TestWorkflowContainerConfig `json:"container,omitempty"`
	Execute    *TestWorkflowStepExecute     `json:"execute,omitempty"`
	Artifacts  *TestWorkflowStepArtifacts   `json:"artifacts,omitempty"`
	Parallel   *TestWorkflowStepParallel    `json:"parallel,omitempty"`
	// nested setup steps to run
	Setup []TestWorkflowStep `json:"setup,omitempty"`
	// nested steps to run
	Steps []TestWorkflowStep `json:"steps,omitempty"`
}
