package commonmapper

import (
	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTargetKubeToAPI(v commonv1.Target) testkube.ExecutionTarget {
	var schedulerPolicy *testkube.SchedulerPolicy
	if v.SchedulerPolicy != "" {
		value := testkube.SchedulerPolicy(v.SchedulerPolicy)
		schedulerPolicy = &value
	}
	return testkube.ExecutionTarget{
		SchedulerPolicy: schedulerPolicy,
		Match:           v.Match,
		Not:             v.Not,
		Replicate:       v.Replicate,
	}
}
