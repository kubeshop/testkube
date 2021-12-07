package detector

import "github.com/kubeshop/testkube/pkg/api/v1/client"

func NewDefaultDetector() Detector {
	d := Detector{}
	d.Add(PostmanCollectionAdapter{})
	d.Add(CurlTestAdapter{})
	return d
}

type Detector struct {
	Adapters []Adapter
}

func (d *Detector) Add(adapter Adapter) {
	d.Adapters = append(d.Adapters, adapter)
}

func (d *Detector) Detect(options client.UpsertScriptOptions) (name string, found bool) {
	for _, adapter := range d.Adapters {
		if name, found := adapter.Is(options); found {
			return name, found
		}
	}

	return
}
