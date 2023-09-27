package deployments

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
)

// MapCRDConditionsToAPI maps Deployment CRD conditions to OpenAPI spec TestTriggerConditions
func MapCRDConditionsToAPI(conditions []appsv1.DeploymentCondition, currentTime time.Time) []testtriggersv1.TestTriggerCondition {
	var results []testtriggersv1.TestTriggerCondition
	for _, condition := range conditions {
		latestTime := condition.LastTransitionTime.Time
		if condition.LastUpdateTime.Time.After(latestTime) {
			latestTime = condition.LastUpdateTime.Time
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
