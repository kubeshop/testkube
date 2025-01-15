package testkube

import "fmt"

func (w *WebhookTemplateCreateRequest) QuoteTextFields() {
	if w.PayloadTemplate != "" {
		w.PayloadTemplate = fmt.Sprintf("%q", w.PayloadTemplate)
	}

	for key, val := range w.Config {
		if val.Value != nil && val.Value.Value != "" {
			val.Value.Value = fmt.Sprintf("%q", val.Value.Value)
		}
		w.Config[key] = val
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
