package testkube

// Clone returns a deep copy. It copies the user too, so a caller that edits the copy
// does not change the original.
func (d *TestWorkflowStatusDetails) Clone() *TestWorkflowStatusDetails {
	if d == nil {
		return nil
	}
	details := *d
	if d.User != nil {
		user := *d.User
		details.User = &user
	}
	return &details
}

// Label returns the layer and the code for a table, and an empty string for an execution that
// passed. A reader of the table sees the kind of failure without the words of the message.
func (d *TestWorkflowStatusDetails) Label() string {
	if d == nil {
		return ""
	}
	if d.Reason == "" {
		return d.Type_
	}
	return d.Type_ + ": " + d.Reason
}
