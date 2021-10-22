package detector

import "github.com/kubeshop/testkube/pkg/api/v1/client"

type Detector struct {
	Adapters []Adapter
}

func (d *Detector) Add(adapter Adapter) {
	d.Adapters = append(d.Adapters, adapter)
}

func (d *Detector) Detect(options client.CreateScriptOptions) string {
	for _, adapter := range d.Adapters {

		if ok, name := adapter.Is(options); ok {
			return name
		}
	}

	return ""
}
