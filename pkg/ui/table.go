package ui

import (
	"encoding/json"
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

type TableData interface {
	Table() (header []string, data [][]string)
}

func (ui *UI) Table(tableData TableData, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
	table.Options(
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off},
			Settings: tw.Settings{
				Lines: tw.Lines{ShowHeaderLine: tw.On},
			},
		}),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
	)
	header, data := tableData.Table()

	if len(header) > 0 {
		anyHeader := make([]any, len(header))
		for i, h := range header {
			anyHeader[i] = h
		}
		table.Header(anyHeader...)
	}

	for _, v := range data {
		table.Append(v) //nolint:errcheck
	}
	table.Render() //nolint:errcheck
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
