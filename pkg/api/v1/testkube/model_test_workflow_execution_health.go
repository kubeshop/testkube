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

type TestWorkflowExecutionHealth struct {
	// Recency-weighted fraction of executions that passed (value between 0.0 and 1.0).
	PassRate float64 `json:"passRate"`
	// Fraction of status changes among consecutive executions without recency weighting  (value between 0.0 and 1.0).
	FlipRate float64 `json:"flipRate"`
	// Combined health score, computed as passRate * (1 - flipRate) (value between 0.0 and 1.0).
	OverallHealth float64 `json:"overallHealth"`
}
