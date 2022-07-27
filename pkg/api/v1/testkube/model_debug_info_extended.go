package testkube

// Table builds up a table from parts of the DebugInfo that are short and easy to read - no logs
func (d DebugInfo) Table() (header []string, output [][]string) {
	header = []string{"Client Version", "Server Version", "Cluster Version"}
	output = make([][]string, 0)

	row := []string{d.ClientVersion, d.ServerVersion, d.ClusterVersion}
	output = append(output, row)

	return header, output
}
