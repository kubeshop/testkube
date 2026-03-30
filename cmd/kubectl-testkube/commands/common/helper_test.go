package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareTestkubeOnPremDemoArgs(t *testing.T) {
	t.Parallel()

	args := prepareTestkubeOnPremDemoArgs(HelmOptions{
		Namespace:     "testkube",
		LicenseKey:    "8A5C9C-7E559A-48745E-B691EF-81A96F-V3",
		DemoValuesURL: "https://example.com/values.demo.yaml",
		SetOptions: map[string]string{
			"testkube-cloud-api.api.agent.host":             "testkube-enterprise-api.testkube.svc.cluster.local",
			"testkube-cloud-api.api.minio.signing.hostname": "testkube-enterprise-minio.testkube.svc.cluster.local:9000",
		},
	})

	settings := demoHelmSettings(args)

	assert.Equal(t, "8A5C9C-7E559A-48745E-B691EF-81A96F-V3", settings["global.enterpriseLicenseKey"])
	assert.Equal(t, "testkube-enterprise-api.testkube.svc.cluster.local", settings["testkube-cloud-api.api.agent.host"])
	assert.Equal(t, "testkube-enterprise-minio.testkube.svc.cluster.local:9000", settings["testkube-cloud-api.api.minio.signing.hostname"])
}

func demoHelmSettings(args []string) map[string]string {
	settings := map[string]string{}

	for i := 0; i < len(args)-1; i++ {
		if args[i] != "--set" {
			continue
		}

		kv := args[i+1]
		for j := 0; j < len(kv); j++ {
			if kv[j] == '=' {
				settings[kv[:j]] = kv[j+1:]
				break
			}
		}
	}

	return settings
}
