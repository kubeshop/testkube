package testkube

func (s *TestWorkflowSignature) Label() string {
	if s.Name != "" {
		return s.Name
	}
	return s.Category
}
