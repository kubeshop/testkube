package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var chartContent = []byte(`
apiVersion: v2
name: testkube
description: A Helm chart for testkube.

# A chart can be either an 'application' or a 'library' chart.
#
# Application charts are a collection of templates that can be packaged into versioned archives
# to be deployed.
#
# Library charts provide useful utilities or functions for the chart developer. They're included as
# a dependency of application charts to inject those utilities and functions into the rendering
# pipeline. Library charts do not define any templates and therefore cannot be deployed.
type: application

# This is the chart version. This version number should be incremented each time you make changes
# to the chart and its templates, including the app version.
# Versions are expected to follow Semantic Versioning (https://semver.org/)
version: 0.5.17 

dependencies:
  - name: testkube-operator
    version: "0.5.7"
    repository: "https://kubeshop.github.io/helm-charts"

  - name: mongodb
    version: "10.0.0"
    repository: "https://charts.bitnami.com/bitnami"

  - name: testkube-api
    version: "0.5.8"
    repository: "https://kubeshop.github.io/helm-charts"
    
  - name: postman-executor
    version: "0.5.7"
    repository: https://kubeshop.github.io/helm-charts
    condition: postman-executor.enabled

  - name: cypress-executor
    version: "0.5.9"
    repository: https://kubeshop.github.io/helm-charts
    condition: cypress-executor.enabled

  - name: curl-executor
    version: "0.5.5"
    repository: https://kubeshop.github.io/helm-charts
    condition: curl-executor.enabled
`)

func TestGetDependencies(t *testing.T) {

	var chart HelmChart
	err := yaml.Unmarshal(chartContent, &chart)
	assert.NoError(t, err)

	t.Run("test GetDependencyVersion", func(t *testing.T) {
		version, err := GetDependencyVersion(chart, "testkube-api")
		assert.NoError(t, err)
		assert.Equal(t, "0.5.8", version)

	})

	t.Run("test UpdateDependencyVersion", func(t *testing.T) {
		chart, err := UpdateDependencyVersion(chart, "testkube-api", "1.2.3")
		assert.NoError(t, err)
		version, err := GetDependencyVersion(chart, "testkube-api")
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3", version)

	})
}
