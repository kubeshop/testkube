package testkube

type Tests []Test

func (t Tests) Table() (header []string, output [][]string) {
	header = []string{"Name", "Type", "Created", "Labels", "Schedule"}
	for _, e := range t {
		output = append(output, []string{
			e.Name,
			e.Type_,
			e.Created.String(),
			MapToString(e.Labels),
			e.Schedule,
		})
	}

	return
}

func (t Test) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name: t.Name,
		// TODO add namespace to test model and all dependencies
		Namespace: "testkube",
	}
}
