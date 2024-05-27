package debugger

import (
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewJobDebugger(name, namespace string) *JobDebugger {
	return &JobDebugger{
		Name:      name,
		Namespace: namespace,
		Writer:    os.Stdout,
		Interval:  time.Second,
	}
}

// JobDebugger is a debugger for jobs
type JobDebugger struct {
	Name      string
	Namespace string
	Writer    io.Writer
	Interval  time.Duration
}

func (d JobDebugger) Run() error {
	for {
		err := d.describePod()
		if err != nil {
			return err
		}
		time.Sleep(d.Interval)
	}
}

func (d *JobDebugger) SetWriter(w io.Writer) {
	d.Writer = w
}

// dummy describePod
func (d JobDebugger) describePod() error {
	ui := ui.NewUI(false, d.Writer)
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}

	args := []string{
		"describe", "pod", "-l", "test-name=" + d.Name, "-n", d.Namespace,
	}

	ui.H1("Pod description time: " + time.Now().String())

	ui.ShellCommand(kubectl, args...)
	ui.NL()

	b, err := process.Execute(kubectl, args...)
	if err != nil {
		return err
	}

	_, err = d.Writer.Write(b)
	return err
}
