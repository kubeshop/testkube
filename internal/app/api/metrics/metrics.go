package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var testExecutionCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_test_executions_count",
	Help: "The total number of test executions",
}, []string{"type", "name", "result"})

var testSuiteExecutionCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testsuite_executions_count",
	Help: "The total number of test suite executions",
}, []string{"name", "result"})

var testCreationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_test_creations_count",
	Help: "The total number of tests created by type events",
}, []string{"type", "result"})

var testSuiteCreationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testsuite_creations_count",
	Help: "The total number of test suites created events",
}, []string{"result"})

var testUpdatesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_test_updates_count",
	Help: "The total number of tests updated by type events",
}, []string{"type", "result"})

var testSuiteUpdatesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testsuite_updates_count",
	Help: "The total number of test suites updated events",
}, []string{"result"})

var testTriggerCreationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testtrigger_creations_count",
	Help: "The total number of test trigger created events",
}, []string{"result"})

var testTriggerUpdatesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testtriggers_updates_count",
	Help: "The total number of test trigger updated events",
}, []string{"result"})

var testTriggerDeletesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testtriggers_deletes_count",
	Help: "The total number of test trigger deleted events",
}, []string{"result"})

var testTriggerBulkUpdatesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testtriggers_bulk_updates_count",
	Help: "The total number of test trigger bulk update events",
}, []string{"result"})

var testTriggerBulkDeletesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testtriggers_bulk_deletes_count",
	Help: "The total number of test trigger bulk delete events",
}, []string{"result"})

var testAbortCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_test_aborts_count",
	Help: "The total number of tests aborted by type events",
}, []string{"type", "result"})

func NewMetrics() Metrics {
	return Metrics{
		TestExecutions:         testExecutionCount,
		TestSuiteExecutions:    testSuiteExecutionCount,
		TestCreations:          testCreationCount,
		TestSuiteCreations:     testSuiteCreationCount,
		TestUpdates:            testUpdatesCount,
		TestSuiteUpdates:       testSuiteUpdatesCount,
		TestTriggerCreations:   testTriggerCreationCount,
		TestTriggerUpdates:     testTriggerUpdatesCount,
		TestTriggerDeletes:     testTriggerDeletesCount,
		TestTriggerBulkUpdates: testTriggerBulkUpdatesCount,
		TestTriggerBulkDeletes: testTriggerBulkDeletesCount,
		TestAbort:              testAbortCount,
	}
}

type Metrics struct {
	TestExecutions         *prometheus.CounterVec
	TestSuiteExecutions    *prometheus.CounterVec
	TestCreations          *prometheus.CounterVec
	TestSuiteCreations     *prometheus.CounterVec
	TestUpdates            *prometheus.CounterVec
	TestSuiteUpdates       *prometheus.CounterVec
	TestTriggerCreations   *prometheus.CounterVec
	TestTriggerUpdates     *prometheus.CounterVec
	TestTriggerDeletes     *prometheus.CounterVec
	TestTriggerBulkUpdates *prometheus.CounterVec
	TestTriggerBulkDeletes *prometheus.CounterVec
	TestAbort              *prometheus.CounterVec
}

func (m Metrics) IncExecuteTest(execution testkube.Execution) {
	status := ""
	if execution.ExecutionResult != nil && execution.ExecutionResult.Status != nil {
		status = string(*execution.ExecutionResult.Status)
	}

	m.TestExecutions.With(map[string]string{
		"type":   execution.TestType,
		"name":   execution.TestName,
		"result": status,
	}).Inc()
}

func (m Metrics) IncExecuteTestSuite(execution testkube.TestSuiteExecution) {
	name := ""
	status := ""
	if execution.TestSuite != nil {
		name = execution.TestSuite.Name
	}

	if execution.Status != nil {
		status = string(*execution.Status)
	}

	m.TestSuiteExecutions.With(map[string]string{
		"name":   name,
		"result": status,
	}).Inc()
}

func (m Metrics) IncUpdateTest(testType string, err error) {
	result := "updated"
	if err != nil {
		result = "error"
	}

	m.TestUpdates.With(map[string]string{
		"type":   testType,
		"result": result,
	}).Inc()
}

func (m Metrics) IncUpdateTestSuite(err error) {
	result := "updated"
	if err != nil {
		result = "error"
	}

	m.TestSuiteUpdates.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncCreateTest(testType string, err error) {
	result := "created"
	if err != nil {
		result = "error"
	}

	m.TestCreations.With(map[string]string{
		"type":   testType,
		"result": result,
	}).Inc()
}

func (m Metrics) IncCreateTestSuite(err error) {
	result := "created"
	if err != nil {
		result = "error"
	}

	m.TestSuiteCreations.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncCreateTestTrigger(err error) {
	result := "created"
	if err != nil {
		result = "error"
	}

	m.TestTriggerCreations.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncUpdateTestTrigger(err error) {
	result := "updated"
	if err != nil {
		result = "error"
	}

	m.TestTriggerUpdates.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncDeleteTestTrigger(err error) {
	result := "deleted"
	if err != nil {
		result = "error"
	}

	m.TestTriggerDeletes.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncBulkUpdateTestTrigger(err error) {
	result := "bulk_update"
	if err != nil {
		result = "error"
	}

	m.TestTriggerBulkUpdates.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncBulkDeleteTestTrigger(err error) {
	result := "bulk_delete"
	if err != nil {
		result = "error"
	}

	m.TestTriggerBulkDeletes.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncAbortTest(testType string, failed bool) {
	result := "aborted"
	if failed {
		result = "error"
	}

	m.TestAbort.With(map[string]string{
		"type":   testType,
		"result": result,
	}).Inc()
}
