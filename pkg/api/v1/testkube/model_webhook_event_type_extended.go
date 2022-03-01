package testkube

func (t *WebhookEventType) String() string {
	return string(*t)
}

func WebhookTypePtr(t WebhookEventType) *WebhookEventType {
	return &t
}

var (
	WebhookTypeStartTest = WebhookTypePtr(START_TEST_WebhookEventType)
	WebhookTypeEndTest   = WebhookTypePtr(END_TEST_WebhookEventType)
)
