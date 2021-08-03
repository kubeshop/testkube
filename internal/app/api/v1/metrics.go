package v1

import (
	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var executionCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kubetest_executions_count",
	Help: "The total number of script executions",
}, []string{"type", "name", "result"})

var creationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kubetest_scripts_creation_count",
	Help: "The total number of scripts created by type events",
}, []string{"type", "result"})

var abortCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kubetest_scripts_abort_count",
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

func (m Metrics) IncExecution(scriptExecution kubetest.ScriptExecution) {
	m.Executions.With(map[string]string{
		"type":   scriptExecution.ScriptType,
		"name":   scriptExecution.ScriptName,
		"result": scriptExecution.Execution.Status,
	}).Inc()
}

func (m Metrics) IncCreateScript(scriptType string, err error) {
	status := "created"
	if err != nil {
		status = "error"
	}

	m.Creations.With(map[string]string{
		"type":   scriptType,
		"status": status,
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
