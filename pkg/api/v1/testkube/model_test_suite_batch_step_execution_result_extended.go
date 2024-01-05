package testkube

import (
	"time"

	"github.com/kubeshop/testkube/pkg/utils"
)

func (e *TestSuiteBatchStepExecutionResult) Start() {
	e.StartTime = time.Now()
}

func (e *TestSuiteBatchStepExecutionResult) Stop() {
	e.EndTime = time.Now()
	e.Duration = utils.RoundDuration(e.CalculateDuration()).String()
}

func (e *TestSuiteBatchStepExecutionResult) CalculateDuration() time.Duration {
	end := e.EndTime
	start := e.StartTime

	if start.UnixNano() <= 0 && end.UnixNano() <= 0 {
		return time.Duration(0)
	}

	if end.UnixNano() <= 0 {
		end = time.Now()
	}

	return end.Sub(e.StartTime)
}
