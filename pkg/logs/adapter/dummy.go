package adapter

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

var _ Adapter = &DebugAdapter{}

// NewDebugAdapter creates new DebugAdapter which will write logs to stdout
func NewDebugAdapter() *DebugAdapter {
	return &DebugAdapter{}
}

type DebugAdapter struct {
	Bucket string
}

func (s *DebugAdapter) Notify(id string, e events.Log) error {
	fmt.Printf("%s %+v\n", id, e)
	return nil
}

func (s *DebugAdapter) Stop(id string) error {
	fmt.Printf("stopping %s \n", id)
	return nil
}

func (s *DebugAdapter) Name() string {
	return "dummy"
}
