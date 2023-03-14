package testkube

import (
	"errors"

	"github.com/adhocore/gronx"
)

func ValidateUpdateTestRequest(test TestUpdateRequest) error {
	if test.Name == nil || *test.Name == "" {
		return errors.New("test name cannot be empty")
	}

	if test.Schedule != nil && *test.Schedule != "" {
		gron := gronx.New()
		if !gron.IsValid(*test.Schedule) {
			return errors.New("invalin cron expression in test schedule")
		}
	}
	return nil
}
