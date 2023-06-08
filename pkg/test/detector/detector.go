package detector

import (
	"github.com/kubeshop/testkube/contrib/executor/artillery/pkg/artillery"
	"github.com/kubeshop/testkube/contrib/executor/curl/pkg/curl"
	"github.com/kubeshop/testkube/contrib/executor/cypress/pkg/cypress"
	"github.com/kubeshop/testkube/contrib/executor/ginkgo/pkg/ginkgo"
	"github.com/kubeshop/testkube/contrib/executor/gradle/pkg/gradle"
	"github.com/kubeshop/testkube/contrib/executor/jmeter/pkg/jmeter"
	"github.com/kubeshop/testkube/contrib/executor/k6/pkg/k6detector"
	"github.com/kubeshop/testkube/contrib/executor/kubepug/pkg/kubepug"
	"github.com/kubeshop/testkube/contrib/executor/maven/pkg/maven"
	"github.com/kubeshop/testkube/contrib/executor/playwright/pkg/playwright"
	"github.com/kubeshop/testkube/contrib/executor/postman/pkg/postman"
	"github.com/kubeshop/testkube/contrib/executor/soapui/pkg/soapui"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
)

func NewDefaultDetector() Detector {
	d := Detector{Adapters: make(map[string]Adapter, 0)}
	d.Add(artillery.Detector{})
	d.Add(curl.Detector{})
	d.Add(jmeter.Detector{})
	d.Add(k6detector.Detector{})
	d.Add(postman.Detector{})
	d.Add(soapui.Detector{})
	d.Add(maven.Detector{})
	d.Add(gradle.Detector{})
	d.Add(playwright.Detector{})
	d.Add(cypress.Detector{})
	d.Add(ginkgo.Detector{})
	d.Add(kubepug.Detector{})
	return d
}

// Detector is detection orchestrator for possible detectors
type Detector struct {
	Adapters map[string]Adapter
}

// Add adds adapter
func (d *Detector) Add(adapter Adapter) {
	d.Adapters[adapter.GetType()] = adapter
}

// Detect detects test type
func (d *Detector) Detect(path string, options client.UpsertTestOptions) (name string, found bool) {
	for _, adapter := range d.Adapters {
		if name, found := adapter.IsWithPath(path, options); found {
			return name, found
		}
	}

	return
}

// GetAdapter return adapter by test type
func (d *Detector) GetAdapter(testType string) Adapter {
	adapter, ok := d.Adapters[testType]
	if !ok {
		return nil
	}

	return adapter
}
