package metrics

import (
	"fmt"
	"slices"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var testExecutionsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_test_executions_count",
	Help: "The total number of test executions",
}, []string{"type", "name", "result", "labels", "test_uri"})

var testExecutionsDurationMs = promauto.NewSummaryVec(prometheus.SummaryOpts{
	Name:       "testkube_test_executions_duration_ms",
	Help:       "The duration of test executions",
	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
}, []string{"type", "name", "result", "labels", "test_uri"})

var testSuiteExecutionsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testsuite_executions_count",
	Help: "The total number of test suite executions",
}, []string{"name", "result", "labels", "testsuite_uri"})

var testSuiteExecutionsDurationMs = promauto.NewSummaryVec(prometheus.SummaryOpts{
	Name:       "testkube_testsuite_executions_duration_ms",
	Help:       "The duration of test suite executions",
	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
}, []string{"name", "result", "labels", "testsuite_uri"})

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

var testWorkflowCreationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflow_creations_count",
	Help: "The total number of test workflow created by type events",
}, []string{"result"})

var testWorkflowUpdatesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflow_updates_count",
	Help: "The total number of test workflow updated by type events",
}, []string{"result"})

var testWorkflowDeletesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflow_deletes_count",
	Help: "The total number of test workflow deleted events",
}, []string{"result"})

var testWorkflowTemplateCreationCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflowtemplate_creations_count",
	Help: "The total number of test workflow template created by type events",
}, []string{"result"})

var testWorkflowTemplateUpdatesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflowtemplate_updates_count",
	Help: "The total number of test workflow template updated by type events",
}, []string{"result"})

var testWorkflowTemplateDeletesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflowtemplate_deletes_count",
	Help: "The total number of test workflow template deleted events",
}, []string{"result"})

func NewMetrics() Metrics {
	return Metrics{
		TestExecutionsCount:           testExecutionsCount,
		TestExecutionsDurationMs:      testExecutionsDurationMs,
		TestSuiteExecutionsCount:      testSuiteExecutionsCount,
		TestSuiteExecutionsDurationMs: testSuiteExecutionsDurationMs,
		TestCreations:                 testCreationCount,
		TestSuiteCreations:            testSuiteCreationCount,
		TestUpdates:                   testUpdatesCount,
		TestSuiteUpdates:              testSuiteUpdatesCount,
		TestTriggerCreations:          testTriggerCreationCount,
		TestTriggerUpdates:            testTriggerUpdatesCount,
		TestTriggerDeletes:            testTriggerDeletesCount,
		TestTriggerBulkUpdates:        testTriggerBulkUpdatesCount,
		TestTriggerBulkDeletes:        testTriggerBulkDeletesCount,
		TestAbort:                     testAbortCount,
		TestWorkflowCreations:         testWorkflowCreationCount,
		TestWorkflowUpdates:           testWorkflowUpdatesCount,
		TestWorkflowDeletes:           testWorkflowDeletesCount,
		TestWorkflowTemplateCreations: testWorkflowTemplateCreationCount,
		TestWorkflowTemplateUpdates:   testWorkflowTemplateUpdatesCount,
		TestWorkflowTemplateDeletes:   testWorkflowTemplateDeletesCount,
	}
}

type Metrics struct {
	TestExecutionsCount           *prometheus.CounterVec
	TestExecutionsDurationMs      *prometheus.SummaryVec
	TestSuiteExecutionsCount      *prometheus.CounterVec
	TestSuiteExecutionsDurationMs *prometheus.SummaryVec
	TestCreations                 *prometheus.CounterVec
	TestSuiteCreations            *prometheus.CounterVec
	TestUpdates                   *prometheus.CounterVec
	TestSuiteUpdates              *prometheus.CounterVec
	TestTriggerCreations          *prometheus.CounterVec
	TestTriggerUpdates            *prometheus.CounterVec
	TestTriggerDeletes            *prometheus.CounterVec
	TestTriggerBulkUpdates        *prometheus.CounterVec
	TestTriggerBulkDeletes        *prometheus.CounterVec
	TestAbort                     *prometheus.CounterVec
	TestWorkflowCreations         *prometheus.CounterVec
	TestWorkflowUpdates           *prometheus.CounterVec
	TestWorkflowDeletes           *prometheus.CounterVec
	TestWorkflowTemplateCreations *prometheus.CounterVec
	TestWorkflowTemplateUpdates   *prometheus.CounterVec
	TestWorkflowTemplateDeletes   *prometheus.CounterVec
}

func (m Metrics) IncAndObserveExecuteTest(execution testkube.Execution, dashboardURI string) {
	status := ""
	if execution.ExecutionResult != nil && execution.ExecutionResult.Status != nil {
		status = string(*execution.ExecutionResult.Status)
	}

	var labels []string
	for key, value := range execution.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}

	slices.Sort(labels)
	m.TestExecutionsCount.With(map[string]string{
		"type":     execution.TestType,
		"name":     execution.TestName,
		"result":   status,
		"labels":   strings.Join(labels, ","),
		"test_uri": fmt.Sprintf("%s/tests/%s", dashboardURI, execution.TestName),
	}).Inc()

	m.TestExecutionsDurationMs.With(map[string]string{
		"type":     execution.TestType,
		"name":     execution.TestName,
		"result":   status,
		"labels":   strings.Join(labels, ","),
		"test_uri": fmt.Sprintf("%s/tests/%s", dashboardURI, execution.TestName),
	}).Observe(float64(execution.DurationMs))
}

func (m Metrics) IncAndObserveExecuteTestSuite(execution testkube.TestSuiteExecution, dashboardURI string) {
	name := ""
	status := ""
	if execution.TestSuite != nil {
		name = execution.TestSuite.Name
	}

	if execution.Status != nil {
		status = string(*execution.Status)
	}

	var labels []string
	for key, value := range execution.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}

	slices.Sort(labels)
	testSuiteName := ""
	if execution.TestSuite != nil {
		testSuiteName = execution.TestSuite.Name
	}

	m.TestSuiteExecutionsCount.With(map[string]string{
		"name":          name,
		"result":        status,
		"labels":        strings.Join(labels, ","),
		"testsuite_uri": fmt.Sprintf("%s/test-suites/%s", dashboardURI, testSuiteName),
	}).Inc()

	m.TestSuiteExecutionsDurationMs.With(map[string]string{
		"name":          name,
		"result":        status,
		"labels":        strings.Join(labels, ","),
		"testsuite_uri": fmt.Sprintf("%s/test-suites/%s", dashboardURI, testSuiteName),
	}).Observe(float64(execution.DurationMs))
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

func (m Metrics) IncCreateTestWorkflow(err error) {
	result := "created"
	if err != nil {
		result = "error"
	}

	m.TestWorkflowCreations.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncUpdateTestWorkflow(err error) {
	result := "updated"
	if err != nil {
		result = "error"
	}

	m.TestWorkflowUpdates.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncDeleteTestWorkflow(err error) {
	result := "deleted"
	if err != nil {
		result = "error"
	}

	m.TestWorkflowDeletes.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncCreateTestWorkflowTemplate(err error) {
	result := "created"
	if err != nil {
		result = "error"
	}

	m.TestWorkflowTemplateCreations.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncUpdateTestWorkflowTemplate(err error) {
	result := "updated"
	if err != nil {
		result = "error"
	}

	m.TestWorkflowTemplateUpdates.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncDeleteTestWorkflowTemplate(err error) {
	result := "deleted"
	if err != nil {
		result = "error"
	}

	m.TestWorkflowTemplateDeletes.With(map[string]string{
		"result": result,
	}).Inc()
}
