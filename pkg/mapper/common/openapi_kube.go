package commonmapper

import (
	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTargetApiToKube(v testkube.ExecutionTarget) commonv1.Target {
	var schedulerPolicy commonv1.SchedulerPolicy
	if v.SchedulerPolicy != nil {
		schedulerPolicy = commonv1.SchedulerPolicy(*v.SchedulerPolicy)
	}
	return commonv1.Target{
		SchedulerPolicy: schedulerPolicy,
		Match:           v.Match,
		Not:             v.Not,
		Replicate:       v.Replicate,
	}
}
