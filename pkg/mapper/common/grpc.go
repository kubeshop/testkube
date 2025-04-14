package commonmapper

import (
	commonv1 "github.com/kubeshop/testkube-operator/api/common/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func MapAllTargetsApiToGrpc(ts []testkube.ExecutionTarget) []*cloud.ExecutionTarget {
	if ts == nil || len(ts) == 0 {
		return nil
	}
	targets := make([]*cloud.ExecutionTarget, len(ts))

	for i, t := range ts {
		target := MapTargetApiToGrpc(&t)
		targets[i] = target
	}

	return targets
}

func MapTargetApiToGrpc(t *testkube.ExecutionTarget) *cloud.ExecutionTarget {
	target := cloud.ExecutionTarget{
		Replicate: t.Replicate,
	}

	if t.Match != nil {
		target.Match = make(map[string]*cloud.ExecutionTargetLabels)
		for k, v := range t.Match {
			target.Match[k] = &cloud.ExecutionTargetLabels{Labels: v}
		}
	}
	if t.Not != nil {
		target.Not = make(map[string]*cloud.ExecutionTargetLabels)
		for k, v := range t.Not {
			target.Not[k] = &cloud.ExecutionTargetLabels{Labels: v}
		}
	}

	return &target
}

func MapAllTargetsKubeToGrpc(ts []commonv1.Target) []*cloud.ExecutionTarget {
	if ts == nil || len(ts) == 0 {
		return nil
	}
	targets := make([]*cloud.ExecutionTarget, len(ts))

	for _, t := range ts {
		target := MapTargetKubeToGrpc(&t)
		targets = append(targets, target)
	}

	return targets
}

func MapTargetKubeToGrpc(t *commonv1.Target) *cloud.ExecutionTarget {
	target := cloud.ExecutionTarget{
		Replicate: t.Replicate,
	}

	if t.Match != nil {
		target.Match = make(map[string]*cloud.ExecutionTargetLabels)
		for k, v := range t.Match {
			target.Match[k] = &cloud.ExecutionTargetLabels{Labels: v}
		}
	}
	if t.Not != nil {
		target.Not = make(map[string]*cloud.ExecutionTargetLabels)
		for k, v := range t.Not {
			target.Not[k] = &cloud.ExecutionTargetLabels{Labels: v}
		}
	}

	return &target
}
