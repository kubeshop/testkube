package debugger

import (
	"io"
	"os"
)

type Debugger interface {
	Run() error
	SetWriter(w io.Writer)
}

func NewInsights() *Insights {
	return &Insights{
		Debuggers: []Debugger{},
		Writer:    os.Stderr,
	}
}

type Insights struct {
	Debuggers []Debugger
	Writer    io.Writer
}

func (i *Insights) WithWriter(w io.Writer) *Insights {
	i.Writer = w
	return i
}

func (i *Insights) AddDebugger(d Debugger) *Insights {
	// set common writer for all debuggers
	d.SetWriter(i.Writer)
	i.Debuggers = append(i.Debuggers, d)
	return i
}

func (i *Insights) Run() *Insights {
	for _, d := range i.Debuggers {
		go func(d Debugger) {
			d.Run()
		}(d)
	}

	return i
}
