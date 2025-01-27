package controller

import (
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

const (
	KubernetesTimezoneLogTimeFormat = constants.PreciseTimeFormat + "07:00"
)

var (
	containerNameRe = regexp.MustCompile(`^spec\.(?:initContainers|containers)\{([^]]+)}`)
)

func GetEventContainerName(event *corev1.Event) string {
	path := event.InvolvedObject.FieldPath
	if containerNameRe.Match([]byte(path)) {
		name := containerNameRe.ReplaceAllString(event.InvolvedObject.FieldPath, "$1")
		return name
	}
	return ""
}

type ContainerResultStep struct {
	Status     testkube.TestWorkflowStepStatus
	ExitCode   int
	Details    string
	FinishedAt time.Time
}

type ContainerResult struct {
	Steps      []ContainerResultStep
	Details    string
	ExitCode   int
	FinishedAt time.Time
}

func ExtractRefsFromActionList(list actiontypes.ActionList) (started []string, finished []string) {
	for i := range list {
		switch list[i].Type() {
		case lite.ActionTypeSetup:
			started = append(started, constants.InitStepName)
			finished = append(finished, constants.InitStepName)
		case lite.ActionTypeStart:
			started = append(started, *list[i].Start)
		case lite.ActionTypeEnd:
			finished = append(finished, *list[i].End)
		}
	}
	return
}

func ExtractRefsFromActionGroup(group actiontypes.ActionGroups) (started [][]string, finished [][]string) {
	started = make([][]string, len(group))
	finished = make([][]string, len(group))
	for i := range group {
		s, f := ExtractRefsFromActionList(group[i])
		started[i] = s
		finished[i] = f
	}
	return
}
