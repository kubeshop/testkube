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

func (s *TestWorkflowSignature) GetParallelStepReference(nameOrReference string) string {
	if s.Category == "Run in parallel" {
		if nameOrReference == "" {
			return s.Ref
		}

		if s.Name == nameOrReference || s.Ref == nameOrReference {
			return s.Ref
		}
	}

	for _, child := range s.Children {
		ref := child.GetParallelStepReference(nameOrReference)
		if ref != "" {
			return ref
		}
	}

	return ""
}
