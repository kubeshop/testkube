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

// A label selector is a label query over a set of resources. The result of matchLabels and matchExpressions are ANDed. An empty label selector matches all objects. A null label selector matches no objects.
type IoK8sApimachineryPkgApisMetaV1LabelSelector struct {
	// matchExpressions is a list of label selector requirements. The requirements are ANDed.
	MatchExpressions []IoK8sApimachineryPkgApisMetaV1LabelSelectorRequirement `json:"matchExpressions,omitempty"`
	// matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}
