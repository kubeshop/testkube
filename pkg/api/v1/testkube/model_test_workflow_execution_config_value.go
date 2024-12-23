/*
 * Testkube API
 *
 * Testkube provides a Kubernetes-native framework for test definition, execution and results
 *
 * API version: 1.0.0
 * Contact: testkube@kubeshop.io
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package testkube

// configuration values used in the test workflow execution
type TestWorkflowExecutionConfigValue struct {
	// configuration value
	Value string `json:"value,omitempty"`
	// configuration value default
	DefaultValue string `json:"defaultValue,omitempty"`
	// indicates if the value is truncated
	Truncated bool `json:"truncated,omitempty"`
}
