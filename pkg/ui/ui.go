// simple ui - TODO use something more sophisticated :)
package ui

import (
	"io"
	"os"
)

var ui = NewStdoutUI(Verbose)

func NewUI(verbose bool, writer io.Writer) *UI {
	return &UI{
		Verbose: verbose,
		Writer:  writer,
	}
}

func NewStdoutUI(verbose bool) *UI {
	return &UI{
		Verbose: verbose,
		Writer:  os.Stdout,
	}
}

type UI struct {
	Verbose bool
	Writer  io.Writer
}

func SetVerbose(verbose bool)                       { ui.Verbose = verbose }
func ExitOnError(item string, errors ...error)      { ui.ExitOnError(item, errors...) }
func PrintOnError(item string, errors ...error)     { ui.PrintOnError(item, errors...) }
func WarnOnError(item string, errors ...error)      { ui.WarnOnError(item, errors...) }
func Logo()                                         { ui.Logo() }
func LogoNoColor()                                  { ui.LogoNoColor() }
func NL()                                           { ui.NL() }
func Success(message string, subMessages ...string) { ui.Success(message, subMessages...) }
func SuccessAndExit(message string, subMessages ...string) {
	ui.SuccessAndExit(message, subMessages...)
}
func Warn(message string, subMessages ...string)  { ui.Warn(message, subMessages...) }
func LogLine(message string)                      { ui.LogLine(message) }
func Debug(message string, subMessages ...string) { ui.Debug(message, subMessages...) }
func Info(message string, subMessages ...string)  { ui.Info(message, subMessages...) }
func Err(err error)                               { ui.Err(err) }
func Errf(err string, params ...interface{})      { ui.Errf(err, params...) }
func Fail(err error)                              { ui.Fail(err) }
func Failf(err string, params ...interface{})     { ui.Failf(err, params...) }
func CommandOutput(output []byte, command string, params ...string) {
	ui.CommandOutput(output, command, params...)
}
func Medal()                                                { ui.Medal() }
func Completed(message string, subMessages ...string)       { ui.Completed(message, subMessages...) }
func GroupCompleted(main string, sub ...string)             { ui.GroupCompleted(main, sub...) }
func InfoGrid(table map[string]string)                      { ui.InfoGrid(table) }
func Vector(table []string)                                 { ui.Vector(table) }
func ShellCommand(title string, commands ...string)         { ui.ShellCommand(title, commands...) }
func Table(tableData TableData, writer io.Writer)           { ui.Table(tableData, writer) }
func JSONTable(tableData TableData, writer io.Writer) error { return ui.JSONTable(tableData, writer) }
func NewArrayTable(a [][]string) ArrayTable                 { return ui.NewArrayTable(a) }
