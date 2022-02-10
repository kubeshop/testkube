package testkube

type Tests []Test

func (scripts Tests) Table() (header []string, output [][]string) {
	header = []string{"Name", "Type"}
	for _, e := range scripts {
		output = append(output, []string{
			e.Name,
			e.Type_,
		})
	}

	return
}

func (s Test) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name: s.Name,
		// TODO add namespace to script model and all dependencies
		Namespace: "testkube",
	}
}
