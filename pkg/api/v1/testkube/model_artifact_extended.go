package testkube

import (
	"strconv"
)

type Artifacts []Artifact

func (artifacts Artifacts) Table() (header []string, output [][]string) {
	header = []string{"Name", "Size (KB)"}
	for _, e := range artifacts {
		output = append(output, []string{
			e.Name,
			strconv.FormatInt(int64(e.Size), 10),
		})
	}

	return
}
