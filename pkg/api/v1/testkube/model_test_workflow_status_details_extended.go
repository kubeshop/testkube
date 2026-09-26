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

// DisplayLabel returns the words for the type and the reason code, for example
// "Infrastructure failure, oom-killed", for text that people read outside the dashboard. Label
// gives the raw codes instead. The reason stays the code, because the code is the stable key. The
// message and the user are never part of it, because the text can leave the product, for example
// to GitHub.
func (d *TestWorkflowStatusDetails) DisplayLabel() string {
	if d == nil {
		return ""
	}
	name := StatusDetailsType(d.Type_).DisplayName()
	switch {
	case name == "":
		return d.Reason
	case d.Reason == "":
		return name
	default:
		return name + ", " + d.Reason
	}
}
