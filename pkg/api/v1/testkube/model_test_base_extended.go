package testkube

type Tests []Test

func (t Tests) Table() (header []string, output [][]string) {
	header = []string{"Name", "Type"}
	for _, e := range t {
		output = append(output, []string{
			e.Name,
			e.Type_,
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
