package localinstall

import _ "embed"

// Bump together: charts and values are tested as a set.
const (
	EnterpriseChart        = "oci://us-east1-docker.pkg.dev/testkube-cloud-372110/testkube/testkube-enterprise"
	EnterpriseChartVersion = "2.337.1"
	RunnerChart            = "oci://us-east1-docker.pkg.dev/testkube-cloud-372110/testkube/testkube-runner"
	RunnerChartVersion     = "2.14.1"
)

// Embedded, so a values change upstream can't break old CLIs.
//
//go:embed values/enterprise-demo.yaml
var EnterpriseDemoValues []byte
