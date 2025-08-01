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

// test suite execution request body
type TestSuiteExecutionRequest struct {
	// test execution custom name
	Name string `json:"name,omitempty"`
	// test suite execution number
	Number int32 `json:"number,omitempty"`
	// test kubernetes namespace (\"testkube\" when not set)
	Namespace string              `json:"namespace,omitempty"`
	Variables map[string]Variable `json:"variables,omitempty"`
	// secret uuid
	SecretUUID string `json:"secretUUID,omitempty"`
	// test suite labels
	Labels map[string]string `json:"labels,omitempty"`
	// execution labels
	ExecutionLabels map[string]string `json:"executionLabels,omitempty"`
	// whether to start execution sync or async
	Sync bool `json:"sync,omitempty"`
	// http proxy for executor containers
	HttpProxy string `json:"httpProxy,omitempty"`
	// https proxy for executor containers
	HttpsProxy string `json:"httpsProxy,omitempty"`
	// duration in seconds the test suite may be active, until its stopped
	Timeout        int32               `json:"timeout,omitempty"`
	ContentRequest *TestContentRequest `json:"contentRequest,omitempty"`
	RunningContext *RunningContext     `json:"runningContext,omitempty"`
	// job template extensions
	JobTemplate string `json:"jobTemplate,omitempty"`
	// name of the template resource
	JobTemplateReference string `json:"jobTemplateReference,omitempty"`
	// cron job template extensions
	CronJobTemplate string `json:"cronJobTemplate,omitempty"`
	// name of the template resource
	CronJobTemplateReference string `json:"cronJobTemplateReference,omitempty"`
	// scraper template extensions
	ScraperTemplate string `json:"scraperTemplate,omitempty"`
	// name of the template resource
	ScraperTemplateReference string `json:"scraperTemplateReference,omitempty"`
	// pvc template extensions
	PvcTemplate string `json:"pvcTemplate,omitempty"`
	// name of the template resource
	PvcTemplateReference string `json:"pvcTemplateReference,omitempty"`
	// number of tests run in parallel
	ConcurrencyLevel int32 `json:"concurrencyLevel,omitempty"`
	// test suite execution name started the test suite execution
	TestSuiteExecutionName string `json:"testSuiteExecutionName,omitempty"`
	// whether webhooks on the execution of this test suite are disabled
	DisableWebhooks bool `json:"disableWebhooks,omitempty"`
}
