/*
 * testkube
 *
 * Efficient testing of k8s applications mandates a k8s native approach to test mgmt/definition/execution - testkube provides a “quality control plane” that natively integrates testing activities into k8s development and operational workflows
 *
 * API version: 0.0.1
 * Contact: api@testkube.io
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package testkube

// environment variable
type KeyValue struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}
