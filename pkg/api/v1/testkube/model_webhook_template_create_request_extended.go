package testkube

func (w *WebhookTemplateCreateRequest) QuoteTextFields() {
	if w.PayloadTemplate != "" {
		w.PayloadTemplate, _ = printPayloadTemplate(w.PayloadTemplate)
	}

	quoteWebhookConfig(w.Config)
	quoteWebhookParameters(w.Parameters)
}
