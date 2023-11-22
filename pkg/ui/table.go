package ui

import (
	"encoding/json"
	"io"

	"github.com/olekukonko/tablewriter"
)

type TableData interface {
	Table() (header []string, data [][]string)
}

func (ui *UI) Table(tableData TableData, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
	table.EnableBorder(false)
	table.SetHeaderLine(true)

	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	header, data := tableData.Table()

	if len(header) > 0 {
		table.SetHeader(header)
	}

	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}

func (ui *UI) JSONTable(tableData TableData, writer io.Writer) error {
	_, data := tableData.Table()
	return json.NewEncoder(writer).Encode(data)
}

func (ui *UI) NewArrayTable(a [][]string) ArrayTable {
	return ArrayTable(a)
}

func (ui *UI) PrintArrayTable(a [][]string) {
	t := ui.NewArrayTable(a)
	ui.Table(t, ui.Writer)
}

type ArrayTable [][]string

func (a ArrayTable) Table() (header []string, data [][]string) {
	return []string{}, a
}
