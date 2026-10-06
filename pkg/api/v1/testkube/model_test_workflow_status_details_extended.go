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

// DisplayLabel returns the display name of the type and the reason code, for example
// "Infrastructure failure, oom-killed", for a short summary such as a CI check. Label gives the
// raw codes instead. The reason stays a code, because the code is the stable key. A reason with
// the same code as the type, such as user-cancel, only repeats the type and is left out.
func (d *TestWorkflowStatusDetails) DisplayLabel() string {
	if d == nil {
		return ""
	}
	name := StatusDetailsType(d.Type_).DisplayName()
	switch {
	case name == "":
		return d.Reason
	case d.Reason == "" || d.Reason == d.Type_:
		return name
	default:
		return name + ", " + d.Reason
	}
}
