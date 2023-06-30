package customresources

import (
	"fmt"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
)

// MapCRDConditionsToAPI maps CRD conditions to OpenAPI spec TestTriggerConditions
func MapCRDConditionsToAPI(conditions map[string]interface{}, currentTime time.Time) []testtriggersv1.TestTriggerCondition {
	// TODO: find a way to generically map conditions to TriggerSpec
	var results []testtriggersv1.TestTriggerCondition
	for _, condition := range conditions {
		c := condition.(map[string]string)

		// TODO: confirm appropriate layout
		layout := "2006-01-02T15:04:05Z"
		t, err := time.Parse(layout, c["LastTransitionTime"])
		if err != nil {
			fmt.Println("Error parsing time:", err)
		}
		status := c["Status"]
		results = append(results, testtriggersv1.TestTriggerCondition{
			Type_:  string(c["Type"]),
			Status: (*testtriggersv1.TestTriggerConditionStatuses)(&status),
			Reason: c["Reason"],
			Ttl:    int32(currentTime.Sub(t) / time.Second),
		})
	}

	return results
}
