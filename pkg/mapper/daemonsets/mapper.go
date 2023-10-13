package daemonsets

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
)

// MapCRDConditionsToAPI maps DaemonSet CRD conditions to OpenAPI spec TestTriggerConditions
func MapCRDConditionsToAPI(conditions []appsv1.DaemonSetCondition, currentTime time.Time) []testtriggersv1.TestTriggerCondition {
	var results []testtriggersv1.TestTriggerCondition
	for _, condition := range conditions {
		results = append(results, testtriggersv1.TestTriggerCondition{
			Type_:  string(condition.Type),
			Status: (*testtriggersv1.TestTriggerConditionStatuses)(&condition.Status),
			Reason: condition.Reason,
			Ttl:    int32(currentTime.Sub(condition.LastTransitionTime.Time) / time.Second),
		})
	}

	return results
}
