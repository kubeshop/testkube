package ui

import (
	"io"

	"github.com/olekukonko/tablewriter"
)

type TableData interface {
	ToArray() (header []string, data [][]string)
}

func Table(tableData TableData, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
	header, data := tableData.ToArray()
	table.SetHeader(header)

	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}
