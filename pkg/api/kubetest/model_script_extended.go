package kubetest

type Scripts []Script

func (scripts Scripts) ToArray() (header []string, output [][]string) {
	header = []string{"Name", "Type", "Created"}
	for _, e := range scripts {
		output = append(output, []string{
			e.Name,
			e.Type_,
			e.Created.String(),
		})
	}

	return
}
