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

type Templates []Template

func (list Templates) Table() (header []string, output [][]string) {
	header = []string{"Name", "Type", "Labels"}

	for _, e := range list {
		templateType := ""
		if e.Type_ != nil {
			templateType = string(*e.Type_)
		}

		output = append(output, []string{
			e.Name,
			templateType,
			MapToString(e.Labels),
		})
	}

	return
}
