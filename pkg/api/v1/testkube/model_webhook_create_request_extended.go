package testkube

func (w *WebhookCreateRequest) QuoteTextFields() {
	if w.PayloadTemplate != "" {
		w.PayloadTemplate, _ = printPayloadTemplate(w.PayloadTemplate)
	}

	quoteWebhookConfig(w.Config)
	quoteWebhookParameters(w.Parameters)
}
