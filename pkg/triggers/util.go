package triggers

import (
	"fmt"
	"strings"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/utils"

	core_v1 "k8s.io/api/core/v1"
)

const testTriggerMaxNameLength = 57

func findContainer(containers []core_v1.Container, target string) *core_v1.Container {
	for _, c := range containers {
		if c.Name == target {
			return &c
		}
	}
	return nil
}

func inPast(t1, t2 time.Time) bool {
	return t1.Before(t2)
}

// GenerateTestTriggerName function generates a trigger name from the TestTrigger spec
// function also takes care of name collisions, not exceeding k8s max object name (63 characters) and not ending with a hyphen '-'
func GenerateTestTriggerName(t *testtriggersv1.TestTrigger) string {
	if t == nil {
		return ""
	}
	name := fmt.Sprintf("trigger-%s-%s-%s-%s", t.Spec.Resource, t.Spec.Event, t.Spec.Action, t.Spec.Execution)
	if len(name) > testTriggerMaxNameLength {
		name = name[:testTriggerMaxNameLength]
	}
	// RFC 1123 compliant names cannot end with a dash
	name = strings.TrimSuffix(name, "-")
	// RFC 1123 compliant names cannot have underscores
	name = strings.ReplaceAll(name, "_", "-")
	name = fmt.Sprintf("%s-%s", name, utils.RandAlphanum(5))
	return name
}
