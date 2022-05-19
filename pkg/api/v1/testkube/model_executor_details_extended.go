package testkube

type ExecutorsDetails []ExecutorDetails

func (list ExecutorsDetails) Table() (header []string, output [][]string) {
	header = []string{"Name", "URI", "Labels"}

	for _, e := range list {
		output = append(output, []string{
			e.Name,
			e.Executor.Uri,
			MapToString(e.Executor.Labels),
		})
	}

	return
}
