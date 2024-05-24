package debugger

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewEventsDebugger(name, namespace string) *EventsDebugger {
	return &EventsDebugger{
		lock:         sync.Mutex{},
		Name:         name,
		Namespace:    namespace,
		Writer:       os.Stdout,
		Interval:     time.Second,
		ExecutionIds: []string{},
	}
}

// EventsDebugger is a debugger for jobs
type EventsDebugger struct {
	lock         sync.Mutex
	Name         string
	Namespace    string
	Writer       io.Writer
	Interval     time.Duration
	ExecutionIds []string
}

func (d *EventsDebugger) Run() error {
	for {
		err := d.getEvents()
		if err != nil {
			return err
		}
		time.Sleep(d.Interval)
	}
}

func (d *EventsDebugger) SetWriter(w io.Writer) {
	d.Writer = w
}

// dummy evnents debug
func (d *EventsDebugger) getEvents() error {
	ui := ui.NewUI(false, d.Writer)
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}
	args := []string{"get", "events", "-A"}
	d.lock.Lock()
	executionIds := d.ExecutionIds
	d.lock.Unlock()

	for _, id := range executionIds {
		ui.H1("Events for execution: " + id + " time: " + time.Now().String())
		events := exec.Command(kubectl, args...)
		grep := exec.Command("grep", id)

		// Get ps's stdout and attach it to grep's stdin.
		pipe, _ := events.StdoutPipe()

		grep.Stdin = pipe

		events.Start()
		res, _ := grep.Output()
		pipe.Close()
		_, err = d.Writer.Write(res)
		if err != nil {
			return err
		}

	}

	return nil
}

func (d *EventsDebugger) AddExecutionId(id string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.ExecutionIds = append(d.ExecutionIds, id)
}
