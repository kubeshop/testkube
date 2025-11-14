package k8sevents

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// TestkubeEventPrefix is prefix for testkube event
const TestkubeEventPrefix = "testkube-event-"

// MapAPIToCRD maps OpenAPI Event spec To CRD Event
func MapAPIToCRD(event testkube.Event, namespace string, eventTime time.Time) corev1.Event {
	var action, reason, message string
	var labels map[string]string

	objectReference := corev1.ObjectReference{
		Kind:      "testkube",
		Name:      "testkube",
		Namespace: namespace,
	}

	if event.TestWorkflowExecution != nil {
		message = fmt.Sprintf("executionId=%s", event.TestWorkflowExecution.Id)
		objectReference.APIVersion = testworkflowsv1.Group + "/" + testworkflowsv1.Version
		objectReference.Kind = testworkflowsv1.Resource
		if event.TestWorkflowExecution.Workflow != nil {
			labels = event.TestWorkflowExecution.Workflow.Labels
			objectReference.Name = event.TestWorkflowExecution.Workflow.Name
		}
	}

	if event.Type_ != nil {
		reason = string(*event.Type_)
		switch *event.Type_ {
		case *testkube.EventStartTestWorkflow:
			action = "started"
		case *testkube.EventEndTestWorkflowSuccess:
			action = "succeed"
		case *testkube.EventEndTestWorkflowFailed:
			action = "failed"
		case *testkube.EventEndTestWorkflowAborted:
			action = "aborted"
		case *testkube.EventQueueTestWorkflow:
			action = "queued"
		case *testkube.EventCreated:
			action = "created"
		case *testkube.EventUpdated:
			action = "updated"
		case *testkube.EventDeleted:
			action = "deleted"
		}
	}

	return corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", TestkubeEventPrefix, event.Id),
			Namespace: namespace,
			Labels:    labels,
		},
		InvolvedObject:      objectReference,
		Action:              action,
		Reason:              reason,
		Message:             message,
		EventTime:           metav1.NewMicroTime(eventTime),
		FirstTimestamp:      metav1.NewTime(eventTime),
		LastTimestamp:       metav1.NewTime(eventTime),
		Type:                "Normal",
		ReportingController: "testkkube.io/services",
		ReportingInstance:   "testkkube.io/services/testkube-api-server",
	}
}
