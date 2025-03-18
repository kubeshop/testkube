package testkube

const parallelCategory = "Run in parallel"

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
	if s.Category == parallelCategory && (nameOrReference == "" || s.Ref == nameOrReference) {
		return s.Ref
	}

	for _, child := range s.Children {
		if s.Name == nameOrReference {
			ref := child.GetParallelStepReference("")
			if ref != "" {
				return ref
			}
		}

		ref := child.GetParallelStepReference(nameOrReference)
		if ref != "" {
			return ref
		}
	}

	return ""
}
