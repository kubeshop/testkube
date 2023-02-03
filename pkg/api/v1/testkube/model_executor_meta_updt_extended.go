package testkube

// IsEmpty check if request is empty
func (e *ExecutorMetaUpdate) IsEmpty() bool {
	if e.IconURI != nil || e.DocsURI != nil || e.Tooltips != nil {
		return false
	}

	return true
}
