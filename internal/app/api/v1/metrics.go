package v1

import (
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var executionCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kubtest_executions_count",
	Help: "The total number of script executions",
}, []string{"type", "name", "result"})

var creationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kubtest_scripts_creation_count",
	Help: "The total number of scripts created by type events",
}, []string{"type", "result"})

var abortCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kubtest_scripts_abort_count",
	Help: "The total number of scripts created by type events",
}, []string{"type", "result"})

func NewMetrics() Metrics {
	return Metrics{
		Executions: executionCount,
		Creations:  creationCount,
		Abort:      abortCount,
	}
}

type Metrics struct {
	Executions *prometheus.CounterVec
	Creations  *prometheus.CounterVec
	Abort      *prometheus.CounterVec
}

func (m Metrics) IncExecution(execution kubtest.Execution) {
	m.Executions.With(map[string]string{
		"type":   execution.ScriptType,
		"name":   execution.ScriptName,
		"result": execution.ExecutionResult.Status,
	}).Inc()
}

func (m Metrics) IncCreateScript(scriptType string, err error) {
	result := "created"
	if err != nil {
		result = "error"
	}

	m.Creations.With(map[string]string{
		"type":   scriptType,
		"result": result,
	}).Inc()
}

func (m Metrics) IncAbortScript(scriptType string, err error) {
	status := "aborted"
	if err != nil {
		status = "error"
	}

	m.Creations.With(map[string]string{
		"type":   scriptType,
		"status": status,
	}).Inc()
}
