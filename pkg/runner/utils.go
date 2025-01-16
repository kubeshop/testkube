package runner

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func retry(count int, delayBase time.Duration, fn func() error) (err error) {
	for i := 0; i < count; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * delayBase)
	}
	return err
}

func buildCloudNotification(streamId string, seqNo uint32, notification testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowNotificationsResponse {
	response := &cloud.TestWorkflowNotificationsResponse{
		StreamId:  streamId,
		SeqNo:     seqNo,
		Timestamp: notification.Ts.Format(time.RFC3339Nano),
		Ref:       notification.Ref,
	}
	if notification.Result != nil {
		m, _ := json.Marshal(notification.Result)
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_RESULT
		response.Message = string(m)
	} else if notification.Output != nil {
		m, _ := json.Marshal(notification.Output)
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_OUTPUT
		response.Message = string(m)
	} else {
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_LOG
		response.Message = notification.Log
	}
	return response
}

func buildCloudError(streamId string, message string) *cloud.TestWorkflowNotificationsResponse {
	ts := time.Now()
	return &cloud.TestWorkflowNotificationsResponse{
		StreamId:  streamId,
		SeqNo:     0,
		Timestamp: ts.Format(time.RFC3339Nano),
		Type:      cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR,
		Message:   fmt.Sprintf("%s %s", ts.Format(constants.PreciseTimeFormat), message),
	}
}

func convertCloudResponseToService(response *cloud.TestWorkflowNotificationsResponse) *cloud.TestWorkflowServiceNotificationsResponse {
	return &cloud.TestWorkflowServiceNotificationsResponse{
		StreamId:  response.StreamId,
		SeqNo:     response.SeqNo,
		Timestamp: response.Timestamp,
		Ref:       response.Ref,
		Type:      response.Type,
		Message:   response.Message,
	}
}

func buildServiceCloudNotification(streamId string, seqNo uint32, notification testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowServiceNotificationsResponse {
	return convertCloudResponseToService(buildCloudNotification(streamId, seqNo, notification))
}

func buildServiceCloudError(streamId string, message string) *cloud.TestWorkflowServiceNotificationsResponse {
	return convertCloudResponseToService(buildCloudError(streamId, message))
}

func convertCloudResponseToParallelStep(response *cloud.TestWorkflowNotificationsResponse) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return &cloud.TestWorkflowParallelStepNotificationsResponse{
		StreamId:  response.StreamId,
		SeqNo:     response.SeqNo,
		Timestamp: response.Timestamp,
		Ref:       response.Ref,
		Type:      response.Type,
		Message:   response.Message,
	}
}

func buildParallelStepCloudNotification(streamId string, seqNo uint32, notification testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return convertCloudResponseToParallelStep(buildCloudNotification(streamId, seqNo, notification))
}

func buildParallelStepCloudError(streamId string, message string) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return convertCloudResponseToParallelStep(buildCloudError(streamId, message))
}
