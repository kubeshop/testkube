package pods

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
)

// MapCRDConditionsToAPI maps Pod CRD conditions to OpenAPI spec TestTriggerConditions
func MapCRDConditionsToAPI(conditions []corev1.PodCondition, currentTime time.Time) []testtriggersv1.TestTriggerCondition {
	var results []testtriggersv1.TestTriggerCondition
	for _, condition := range conditions {
		latestTime := condition.LastTransitionTime.Time
		if condition.LastProbeTime.Time.After(latestTime) {
			latestTime = condition.LastProbeTime.Time
		}

		results = append(results, testtriggersv1.TestTriggerCondition{
			Type_:  string(condition.Type),
			Status: (*testtriggersv1.TestTriggerConditionStatuses)(&condition.Status),
			Reason: condition.Reason,
			Ttl:    int32(currentTime.Sub(latestTime) / time.Second),
		})
	}

	return results
}
