package customresources

import (
	"fmt"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
)

// MapCRDConditionsToAPI maps CRD conditions to OpenAPI spec TestTriggerConditions
func MapCRDConditionsToAPI(conditions []interface{}, currentTime time.Time) []testtriggersv1.TestTriggerCondition {
	// TODO: find a way to generically map conditions to TriggerSpec
	var results []testtriggersv1.TestTriggerCondition
	for _, condition := range conditions {
		c := make(map[string]string)
		for key, value := range condition.(map[string]interface{}) {
			strKey := fmt.Sprintf("%v", key)
			strValue := fmt.Sprintf("%v", value)

			c[strKey] = strValue
		}

		// TODO: confirm appropriate layout
		layout := time.RFC3339
		t, err := time.Parse(layout, c["lastTransitionTime"])
		if err != nil {
			fmt.Println("Error parsing time:", err)
		}
		status := c["status"]
		results = append(results, testtriggersv1.TestTriggerCondition{
			Type_:  string(c["type"]),
			Status: (*testtriggersv1.TestTriggerConditionStatuses)(&status),
			Reason: c["reason"],
			Ttl:    int32(currentTime.Sub(t) / time.Second),
		})
	}

	return results
}
