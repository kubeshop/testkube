package commonmapper

import (
	commonv1 "github.com/kubeshop/testkube-operator/api/common/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTargetKubeToAPI(v commonv1.Target) testkube.ExecutionTarget {
	return testkube.ExecutionTarget{
		Match:     v.Match,
		Not:       v.Not,
		Replicate: v.Replicate,
	}
}
