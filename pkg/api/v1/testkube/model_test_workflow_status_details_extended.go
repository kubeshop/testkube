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
