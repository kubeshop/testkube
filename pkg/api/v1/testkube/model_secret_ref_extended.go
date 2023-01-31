package testkube

// IsEmpty check if secret ref is empty
func (s *SecretRef) IsEmpty() bool {
	if s.Namespace != "" || s.Name != "" || s.Key != "" {
		return false
	}

	return true
}
