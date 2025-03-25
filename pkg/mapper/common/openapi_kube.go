package commonmapper

import (
	commonv1 "github.com/kubeshop/testkube-operator/api/common/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTargetApiToKube(v testkube.ExecutionTarget) commonv1.Target {
	return commonv1.Target{
		Match:     v.Match,
		Not:       v.Not,
		Replicate: v.Replicate,
	}
}
