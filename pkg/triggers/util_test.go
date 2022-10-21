package triggers

import (
	"testing"

	v1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTestTriggerName(t *testing.T) {
	testTrigger1 := v1.TestTrigger{
		Spec: v1.TestTriggerSpec{
			Resource:  "deployment",
			Event:     "deployment_scale_modified",
			Action:    "run",
			Execution: "test",
		}}
	expected1 := "trigger-deployment-deployment-scale-modified-run-test-"
	actual1 := GenerateTestTriggerName(&testTrigger1)
	assert.Contains(t, actual1, expected1)

	testTrigger2 := v1.TestTrigger{
		Spec: v1.TestTriggerSpec{
			Resource:  "pod",
			Event:     "some-really-really-really-really-really-really-really-really-really-really-really-really-really-really-really-long-event",
			Action:    "run",
			Execution: "test",
		}}
	actual2 := GenerateTestTriggerName(&testTrigger2)
	assert.Len(t, actual2, 63)

	testTrigger3 := v1.TestTrigger{
		Spec: v1.TestTriggerSpec{
			Resource:  "service",
			Event:     "created",
			Action:    "run",
			Execution: "test",
		}}
	expected3 := "trigger-service-created-run-test-"
	actual3 := GenerateTestTriggerName(&testTrigger3)
	assert.Contains(t, actual3, expected3)

	actual4 := GenerateTestTriggerName(nil)
	assert.Empty(t, actual4)
}
