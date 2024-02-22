// simple ui - TODO use something more sophisticated :)
package ui

import (
	"io"
	"os"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/internal/common"
)

const (
	Separator = "separator"
)

var (
	uiOut = NewStdoutUI(Verbose)
	uiErr = NewStderrUI(Verbose)
	ui    = uiOut
)

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

func NewStderrUI(verbose bool) *UI {
	return &UI{
		Verbose: verbose,
		Writer:  os.Stderr,
	}
}

type UI struct {
	Verbose bool
	Writer  io.Writer
}

func SetVerbose(verbose bool)                       { ui.Verbose = verbose }
func IsVerbose() bool                               { return ui.Verbose }
func ExitOnError(item string, errors ...error)      { ui.ExitOnError(item, errors...) }
func PrintOnError(item string, errors ...error)     { ui.PrintOnError(item, errors...) }
func WarnOnError(item string, errors ...error)      { ui.WarnOnError(item, errors...) }
func Logo()                                         { ui.Logo() }
func LogoNoColor()                                  { ui.LogoNoColor() }
func NL(amount ...int)                              { ui.NL(amount...) }
func H1(message string)                             { ui.H1(message) }
func H2(message string)                             { ui.H2(message) }
func Paragraph(message string)                      { ui.Paragraph(message) }
func Success(message string, subMessages ...string) { ui.Success(message, subMessages...) }
func SuccessAndExit(message string, subMessages ...string) {
	ui.SuccessAndExit(message, subMessages...)
}
func Warn(message string, subMessages ...string)  { ui.Warn(message, subMessages...) }
func Alert(message string, subMessages ...string) { ui.Alert(message, subMessages...) }
func LogLine(message string)                      { ui.LogLine(message) }
func Debug(message string, subMessages ...string) { ui.Debug(message, subMessages...) }
func Info(message string, subMessages ...string)  { ui.Info(message, subMessages...) }
func Link(message string, subMessages ...string)  { ui.Link(message, subMessages...) }
func Err(err error)                               { ui.Err(err) }
func Errf(err string, params ...interface{})      { ui.Errf(err, params...) }
func Fail(err error)                              { ui.Fail(err) }
func Failf(err string, params ...interface{})     { ui.Failf(err, params...) }
func CommandOutput(output []byte, command string, params ...string) {
	ui.CommandOutput(output, command, params...)
}
func Print(message string, subMessages ...string)           { ui.Print(message, subMessages...) }
func Printf(format string, data ...any)                     { ui.Printf(format, data...) }
func PrintDot()                                             { ui.PrintDot() }
func PrintEnabled(message string, subMessages ...string)    { ui.PrintEnabled(message, subMessages...) }
func PrintDisabled(message string, subMessages ...string)   { ui.PrintDisabled(message, subMessages...) }
func Medal()                                                { ui.Medal() }
func Completed(message string, subMessages ...string)       { ui.Completed(message, subMessages...) }
func GroupCompleted(main string, sub ...string)             { ui.GroupCompleted(main, sub...) }
func InfoGrid(table map[string]string)                      { ui.InfoGrid(table) }
func Properties(table [][]string)                           { ui.Properties(table) }
func Vector(table []string)                                 { ui.Vector(table) }
func ShellCommand(title string, commands ...string)         { ui.ShellCommand(title, commands...) }
func Table(tableData TableData, writer io.Writer)           { ui.Table(tableData, writer) }
func JSONTable(tableData TableData, writer io.Writer) error { return ui.JSONTable(tableData, writer) }
func NewArrayTable(a [][]string) ArrayTable                 { return ui.NewArrayTable(a) }
func PrintArrayTable(a [][]string)                          { ui.PrintArrayTable(a) }
func Confirm(message string) bool                           { return ui.Confirm(message) }
func Select(title string, options []string) string          { return ui.Select(title, options) }
func TextInput(message string) string                       { return ui.TextInput(message) }

func PrintCRD[T interface{}](cr T, kind string, groupVersion schema.GroupVersion) {
	PrintCRDs([]T{cr}, kind, groupVersion)
}

func PrintCRDs[T interface{}](crs []T, kind string, groupVersion schema.GroupVersion) {
	bytes, err := common.SerializeCRDs(crs, common.SerializeOptions{
		OmitCreationTimestamp: true,
		CleanMeta:             true,
		Kind:                  kind,
		GroupVersion:          &groupVersion,
	})
	ui.ExitOnError("serializing the crds", err)
	_, _ = os.Stdout.Write(bytes)
}

func UseStdout() { ui = uiOut }
func UseStderr() { ui = uiErr }
