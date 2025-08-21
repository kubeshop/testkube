package metrics

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows"
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

var testAbortCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_test_aborts_count",
	Help: "The total number of tests aborted by type events",
}, []string{"type", "result"})

var testSuiteExecutionsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testsuite_executions_count",
	Help: "The total number of test suite executions",
}, []string{"name", "result", "labels", "testsuite_uri"})

var testSuiteExecutionsDurationMs = promauto.NewSummaryVec(prometheus.SummaryOpts{
	Name:       "testkube_testsuite_executions_duration_ms",
	Help:       "The duration of test suite executions",
	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
}, []string{"name", "result", "labels", "testsuite_uri"})

var testSuiteAbortCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testsuite_aborts_count",
	Help: "The total number of test suites aborted by type events",
}, []string{"result"})

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

var testWorkflowExecutionsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflow_executions_count",
	Help: "The total number of test workflow executions",
}, []string{"name", "result", "labels", "testworkflow_uri", "triggered_by", "tags"})

var testWorkflowExecutionsDurationMs = promauto.NewSummaryVec(prometheus.SummaryOpts{
	Name:       "testkube_testworkflow_executions_duration_ms",
	Help:       "The duration of test workflow executions",
	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
}, []string{"name", "result", "labels", "testworkflow_uri", "triggered_by", "tags"})

var testWorkflowAbortCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflow_aborts_count",
	Help: "The total number of test workflows aborted by type events",
}, []string{"result"})

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

var testTriggerEventCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testtrigger_event_count",
	Help: "The total number of test trigger events",
}, []string{"name", "resource", "eventType", "causes"})

var webhookExecutionsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_webhook_executions_count",
	Help: "The total number of webhook executions",
}, []string{"name", "eventType", "result"})

var testWorkflowExecutionStepsDurationMs = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "testkube_testworkflow_execution_steps_duration_ms",
	Help: "The duration of test workflow execution steps",
}, []string{"workflow_name", "step_name"})

var testWorkflowExecutionStepsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "testkube_testworkflow_execution_steps_count",
	Help: "The total number of test workflow execution steps",
}, []string{"workflow_name", "step_name", "status"})

var testWorkflowExecutionStepsStartTimeMs = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "testkube_testworkflow_execution_steps_start_time_ms",
	Help: "The start time of test workflow execution steps",
}, []string{"workflow_name", "step_name", "status"})

var testWorkflowExecutionStepsFinishTimeMs = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "testkube_testworkflow_execution_steps_finish_time_ms",
	Help: "The finish time of test workflow execution steps",
}, []string{"workflow_name", "step_name", "status"})

func NewMetrics() Metrics {
	return Metrics{
		TestExecutionsCount:                    testExecutionsCount,
		TestExecutionsDurationMs:               testExecutionsDurationMs,
		TestAbort:                              testAbortCount,
		TestSuiteExecutionsCount:               testSuiteExecutionsCount,
		TestSuiteExecutionsDurationMs:          testSuiteExecutionsDurationMs,
		TestSuiteAbort:                         testSuiteAbortCount,
		TestCreations:                          testCreationCount,
		TestSuiteCreations:                     testSuiteCreationCount,
		TestUpdates:                            testUpdatesCount,
		TestSuiteUpdates:                       testSuiteUpdatesCount,
		TestTriggerCreations:                   testTriggerCreationCount,
		TestTriggerUpdates:                     testTriggerUpdatesCount,
		TestTriggerDeletes:                     testTriggerDeletesCount,
		TestTriggerBulkUpdates:                 testTriggerBulkUpdatesCount,
		TestTriggerBulkDeletes:                 testTriggerBulkDeletesCount,
		TestWorkflowExecutionsCount:            testWorkflowExecutionsCount,
		TestWorkflowExecutionsDurationMs:       testWorkflowExecutionsDurationMs,
		TestWorkflowAbort:                      testWorkflowAbortCount,
		TestWorkflowCreations:                  testWorkflowCreationCount,
		TestWorkflowUpdates:                    testWorkflowUpdatesCount,
		TestWorkflowDeletes:                    testWorkflowDeletesCount,
		TestWorkflowTemplateCreations:          testWorkflowTemplateCreationCount,
		TestWorkflowTemplateUpdates:            testWorkflowTemplateUpdatesCount,
		TestWorkflowTemplateDeletes:            testWorkflowTemplateDeletesCount,
		TestTriggerEventCount:                  testTriggerEventCount,
		WebhookEventCount:                      webhookExecutionsCount,
		TestWorkflowExecutionStepsDurationMs:   testWorkflowExecutionStepsDurationMs,
		TestWorkflowExecutionStepsCount:        testWorkflowExecutionStepsCount,
		TestWorkflowExecutionStepsStartTimeMs:  testWorkflowExecutionStepsStartTimeMs,
		TestWorkflowExecutionStepsFinishTimeMs: testWorkflowExecutionStepsFinishTimeMs,
	}
}

type Metrics struct {
	TestExecutionsCount                    *prometheus.CounterVec
	TestExecutionsDurationMs               *prometheus.SummaryVec
	TestAbort                              *prometheus.CounterVec
	TestSuiteExecutionsCount               *prometheus.CounterVec
	TestSuiteExecutionsDurationMs          *prometheus.SummaryVec
	TestSuiteAbort                         *prometheus.CounterVec
	TestCreations                          *prometheus.CounterVec
	TestSuiteCreations                     *prometheus.CounterVec
	TestUpdates                            *prometheus.CounterVec
	TestSuiteUpdates                       *prometheus.CounterVec
	TestTriggerCreations                   *prometheus.CounterVec
	TestTriggerUpdates                     *prometheus.CounterVec
	TestTriggerDeletes                     *prometheus.CounterVec
	TestTriggerBulkUpdates                 *prometheus.CounterVec
	TestTriggerBulkDeletes                 *prometheus.CounterVec
	TestWorkflowExecutionsCount            *prometheus.CounterVec
	TestWorkflowExecutionsDurationMs       *prometheus.SummaryVec
	TestWorkflowAbort                      *prometheus.CounterVec
	TestWorkflowCreations                  *prometheus.CounterVec
	TestWorkflowUpdates                    *prometheus.CounterVec
	TestWorkflowDeletes                    *prometheus.CounterVec
	TestWorkflowTemplateCreations          *prometheus.CounterVec
	TestWorkflowTemplateUpdates            *prometheus.CounterVec
	TestWorkflowTemplateDeletes            *prometheus.CounterVec
	TestTriggerEventCount                  *prometheus.CounterVec
	WebhookEventCount                      *prometheus.CounterVec
	TestWorkflowExecutionStepsDurationMs   *prometheus.GaugeVec
	TestWorkflowExecutionStepsCount        *prometheus.CounterVec
	TestWorkflowExecutionStepsStartTimeMs  *prometheus.GaugeVec
	TestWorkflowExecutionStepsFinishTimeMs *prometheus.GaugeVec
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

	m.TestSuiteExecutionsCount.With(map[string]string{
		"name":          name,
		"result":        status,
		"labels":        strings.Join(labels, ","),
		"testsuite_uri": fmt.Sprintf("%s/test-suites/%s", dashboardURI, name),
	}).Inc()

	m.TestSuiteExecutionsDurationMs.With(map[string]string{
		"name":          name,
		"result":        status,
		"labels":        strings.Join(labels, ","),
		"testsuite_uri": fmt.Sprintf("%s/test-suites/%s", dashboardURI, name),
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

func (m Metrics) IncAbortTestSuite() {
	result := "aborted"
	m.TestSuiteAbort.With(map[string]string{
		"result": result,
	}).Inc()
}

func (m Metrics) IncAndObserveExecuteTestWorkflow(execution testkube.TestWorkflowExecution, dashboardURI string) {
	name := ""
	status := ""
	if execution.Workflow != nil {
		name = execution.Workflow.Name
	}

	if execution.Result != nil && execution.Result.Status != nil {
		status = string(*execution.Result.Status)
	}

	var labels []string
	if execution.Workflow != nil {
		for key, value := range execution.Workflow.Labels {
			labels = append(labels, fmt.Sprintf("%s=%s", key, value))
		}
	}

	slices.Sort(labels)

	triggeredBy := ""
	if execution.RunningContext != nil && execution.RunningContext.Actor != nil {
		triggeredBy = string(*execution.RunningContext.Actor.Type_)
		if execution.RunningContext.Actor.Name != "" {
			if triggeredBy != "" {
				triggeredBy += ","
			}

			triggeredBy += execution.RunningContext.Actor.Name
		}
	}

	var tags []string
	for key, value := range execution.Tags {
		tags = append(tags, fmt.Sprintf("%s=%s", key, value))
	}

	slices.Sort(tags)

	m.TestWorkflowExecutionsCount.With(map[string]string{
		"name":             name,
		"result":           status,
		"labels":           strings.Join(labels, ","),
		"testworkflow_uri": fmt.Sprintf("%s/test-workflows/%s", dashboardURI, name),
		"triggered_by":     triggeredBy,
		"tags":             strings.Join(tags, ","),
	}).Inc()

	if execution.Result != nil {
		m.TestWorkflowExecutionsDurationMs.With(map[string]string{
			"name":             name,
			"result":           status,
			"labels":           strings.Join(labels, ","),
			"testworkflow_uri": fmt.Sprintf("%s/test-workflows/%s", dashboardURI, name),
			"triggered_by":     triggeredBy,
			"tags":             strings.Join(tags, ","),
		}).Observe(float64(execution.Result.DurationMs))

		if execution.Result.Steps != nil {
			steps := testworkflows.FlattenSignatures(execution.Signature)
			for _, step := range steps {
				var duration time.Duration
				var status *testkube.TestWorkflowStepStatus
				var startTime, finishTime time.Time
				if result, ok := execution.Result.Steps[step.Ref]; ok {
					duration = max(result.FinishedAt.Sub(result.QueuedAt), 0)
					status = result.Status
					startTime = result.QueuedAt
					finishTime = result.FinishedAt
				}

				m.TestWorkflowExecutionStepsDurationMs.With(map[string]string{
					"workflow_name": name,
					"step_name":     step.Label(),
				}).Set(float64(duration.Milliseconds()))

				if status != nil {
					m.TestWorkflowExecutionStepsCount.With(map[string]string{
						"workflow_name": name,
						"step_name":     step.Label(),
						"status":        string(*status),
					}).Inc()

					m.TestWorkflowExecutionStepsStartTimeMs.With(map[string]string{
						"workflow_name": name,
						"step_name":     step.Label(),
						"status":        string(*status),
					}).Set(float64(startTime.UnixMilli()))

					m.TestWorkflowExecutionStepsFinishTimeMs.With(map[string]string{
						"workflow_name": name,
						"step_name":     step.Label(),
						"status":        string(*status),
					}).Set(float64(finishTime.UnixMilli()))
				}
			}
		}
	}
}

func (m Metrics) IncAbortTestWorkflow() {
	result := "aborted"
	m.TestWorkflowAbort.With(map[string]string{
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

func (m Metrics) IncTestTriggerEventCount(name, resource, eventType string, causes []string) {
	slices.Sort(causes)
	m.TestTriggerEventCount.With(map[string]string{
		"name":      name,
		"resource":  resource,
		"eventType": eventType,
		"causes":    strings.Join(causes, ","),
	}).Inc()
}

func (m Metrics) IncWebhookEventCount(name, eventType, result string) {
	m.WebhookEventCount.With(map[string]string{
		"name":      name,
		"eventType": eventType,
		"result":    result,
	}).Inc()
}
