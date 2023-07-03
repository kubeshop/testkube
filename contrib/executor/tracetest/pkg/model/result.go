package model

import (
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	PASSED_TEST_ICON = "✔"
	FAILED_TEST_ICON = "✘"
)

type Result struct {
	Output         string
	ServerEndpoint string
	OutputEndpoint string
}

func (r *Result) GetOutput() string {
	if r.OutputEndpoint != "" {
		return strings.ReplaceAll(r.Output, r.ServerEndpoint, r.OutputEndpoint)
	}
	return r.Output
}

func (r *Result) GetStatus() *testkube.ExecutionStatus {
	if r.IsSuccessful() {
		return testkube.ExecutionStatusPassed
	}

	return testkube.ExecutionStatusFailed
}

func (r *Result) IsSuccessful() bool {
	return !strings.Contains(r.Output, FAILED_TEST_ICON)
}
