package v1

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var executionCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_executions_count",
	Help: "The total number of test executions",
}, []string{"type", "name", "result"})

var creationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_tests_creation_count",
	Help: "The total number of tests created by type events",
}, []string{"type", "result"})

var updatesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_tests_updates_count",
	Help: "The total number of tests created by type events",
}, []string{"type", "result"})

var abortCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_tests_abort_count",
	Help: "The total number of tests created by type events",
}, []string{"type", "result"})

func NewMetrics() Metrics {
	return Metrics{
		Executions: executionCount,
		Creations:  creationCount,
		Updates:    updatesCount,
		Abort:      abortCount,
	}
}

type Metrics struct {
	Executions *prometheus.CounterVec
	Creations  *prometheus.CounterVec
	Updates    *prometheus.CounterVec
	Abort      *prometheus.CounterVec
}

func (m Metrics) IncExecution(execution testkube.Execution) {
	m.Executions.With(map[string]string{
		"type":   execution.TestType,
		"name":   execution.TestName,
		"result": string(*execution.ExecutionResult.Status),
	}).Inc()
}

func (m Metrics) IncUpdateTest(testType string, err error) {
	result := "updated"
	if err != nil {
		result = "error"
	}

	m.Updates.With(map[string]string{
		"type":   testType,
		"result": result,
	}).Inc()
}

func (m Metrics) IncCreateTest(testType string, err error) {
	result := "created"
	if err != nil {
		result = "error"
	}

	m.Creations.With(map[string]string{
		"type":   testType,
		"result": result,
	}).Inc()
}

func (m Metrics) IncAbortTest(testType string, err error) {
	status := "aborted"
	if err != nil {
		status = "error"
	}

	m.Creations.With(map[string]string{
		"type":   testType,
		"status": status,
	}).Inc()
}
