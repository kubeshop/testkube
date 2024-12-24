package testkube

import "fmt"

func (w *WebhookCreateRequest) QuoteTextFields() {
	if w.PayloadTemplate != "" {
		w.PayloadTemplate = fmt.Sprintf("%q", w.PayloadTemplate)
	}

	for key, value := range w.Config {
		if value.Public != nil && value.Public.Value != "" {
			value.Public.Value = fmt.Sprintf("%q", value.Public.Value)
		}
		w.Config[key] = value
	}

	for key, value := range w.Parameters {
		if value.Description != "" {
			value.Description = fmt.Sprintf("%q", value.Description)
		}

		if value.Example != "" {
			value.Example = fmt.Sprintf("%q", value.Example)
		}

		if value.Default_ != nil && value.Default_.Value != "" {
			value.Pattern = fmt.Sprintf("%q", value.Pattern)
		}

		if value.Pattern != "" {
			value.Pattern = fmt.Sprintf("%q", value.Pattern)
		}

		w.Parameters[key] = value
	}
}
