package testkube

// Cause is a reason code with the text that Kubernetes reported for it.
type Cause struct {
	// Reason is a StopReason or a StartReason code.
	Reason string
	// Message is the text that Kubernetes reported, empty when it reported none.
	Message string
}

// String returns the words for the reason and the reported text.
// The text has no color codes, because the execution result stores it.
func (c Cause) String() string {
	words := StopReason(c.Reason).Sentence()
	if words == "" {
		words = StartReason(c.Reason).Sentence()
	}
	if words == "" {
		words = c.Reason
	}
	if c.Message == "" {
		return words
	}
	return words + ": " + c.Message
}
