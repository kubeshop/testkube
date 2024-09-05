package testkube

func (s *TestWorkflowSignature) Label() string {
	if s.Name != "" {
		return s.Name
	}
	return s.Category
}

func (s *TestWorkflowSignature) Sequence() []TestWorkflowSignature {
	result := []TestWorkflowSignature{*s}
	for i := range s.Children {
		result = append(result, s.Children[i].Sequence()...)
	}
	return result
}
