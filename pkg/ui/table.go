package ui

import (
	"encoding/json"
	"io"

	"github.com/olekukonko/tablewriter"
)

type TableData interface {
	Table() (header []string, data [][]string)
}

func Table(tableData TableData, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
	table.SetBorder(false)
	header, data := tableData.Table()

	if len(header) > 0 {
		table.SetHeader(header)
	}

	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}

func JSONTable(tableData TableData, writer io.Writer) error {
	_, data := tableData.Table()
	return json.NewEncoder(writer).Encode(data)
}

func NewArrayTable(a [][]string) ArrayTable {
	return ArrayTable(a)
}

type ArrayTable [][]string

func (a ArrayTable) Table() (header []string, data [][]string) {
	return []string{}, a
}
