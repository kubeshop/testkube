package detector

import "github.com/kubeshop/testkube/pkg/api/v1/client"

func NewDefaultDetector() Detector {
	d := Detector{Adapters: make(map[string]Adapter, 0)}
	d.Add(PostmanCollectionAdapter{})
	d.Add(CurlTestAdapter{})
	d.Add(K6Adapter{})
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
func (d *Detector) Detect(options client.UpsertTestOptions) (name string, found bool) {
	for _, adapter := range d.Adapters {
		if name, found := adapter.Is(options); found {
			return name, found
		}
	}

	return
}

// DetectTestName detects test name
func (d *Detector) DetectTestName(filename string) (name, testType string, found bool) {
	for _, adapter := range d.Adapters {
		if name, found := adapter.IsTestName(filename); found {
			return name, adapter.GetType(), found
		}
	}

	return
}

// DetectEnvName detects env name
func (d *Detector) DetectEnvName(filename string) (name, env, testType string, found bool) {
	for _, adapter := range d.Adapters {
		if name, env, found := adapter.IsEnvName(filename); found {
			return name, env, adapter.GetType(), found
		}
	}

	return
}

// DetectSecretEnvName detecs secret env name
func (d *Detector) DetectSecretEnvName(filename string) (name, env, testType string, found bool) {
	for _, adapter := range d.Adapters {
		if name, env, found := adapter.IsSecretEnvName(filename); found {
			return name, env, adapter.GetType(), found
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
