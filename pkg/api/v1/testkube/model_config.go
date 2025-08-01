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

// Testkube API config data structure
type Config struct {
	Id              string `json:"id"`
	ClusterId       string `json:"clusterId"`
	EnableTelemetry bool   `json:"enableTelemetry"`
}
