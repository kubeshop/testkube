package testworkflowprocessor

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/rand"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

var BypassToolkitCheck = corev1.EnvVar{
	Name:  "TK_TC_SECURITY",
	Value: rand.String(20),
}

func MapResourcesToKubernetesResources(resources *testworkflowsv1.Resources) (corev1.ResourceRequirements, error) {
	result := corev1.ResourceRequirements{}
	if resources != nil {
		if len(resources.Requests) > 0 {
			result.Requests = make(corev1.ResourceList)
		}
		if len(resources.Limits) > 0 {
			result.Limits = make(corev1.ResourceList)
		}
		for k, v := range resources.Requests {
			var err error
			result.Requests[k], err = quantity.ParseQuantity(v.String())
			if err != nil {
				return corev1.ResourceRequirements{}, errors.Wrap(err, "parsing resources")
			}
		}
		for k, v := range resources.Limits {
			var err error
			result.Limits[k], err = quantity.ParseQuantity(v.String())
			if err != nil {
				return corev1.ResourceRequirements{}, errors.Wrap(err, "parsing resources")
			}
		}
	}
	return result, nil
}

func AnnotateControlledBy(obj metav1.Object, rootId, id string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constants.RootResourceIdLabelName] = rootId
	labels[constants.ResourceIdLabelName] = id
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateControlledBy(&v.Spec.Template, rootId, id)
	}
}

func AnnotateGroupId(obj metav1.Object, id string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constants.GroupIdLabelName] = id
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateGroupId(&v.Spec.Template, id)
	}
}

func getRef(stage Stage) string {
	return stage.Ref()
}

func isNotOptional(stage Stage) bool {
	return !stage.Optional()
}

func buildKubernetesContainers(stage Stage, init *initProcess, fsGroup *int64, machines ...expressions.Machine) (containers []corev1.Container, err error) {
	if stage.Paused() {
		init.SetPaused(stage.Paused())
	}
	if stage.Timeout() != "" {
		init.AddTimeout(stage.Timeout(), stage.Ref())
	}
	if stage.Ref() != "" {
		init.AddCondition(stage.Condition(), stage.Ref())
	}
	init.AddRetryPolicy(stage.RetryPolicy(), stage.Ref())

	group, ok := stage.(GroupStage)
	if ok {
		recursiveRefs := common.MapSlice(group.RecursiveChildren(), getRef)
		directRefResults := common.MapSlice(common.FilterSlice(group.Children(), isNotOptional), getRef)

		init.AddCondition(stage.Condition(), recursiveRefs...)

		if group.Negative() {
			// Create virtual layer that will be put down into actual negative step
			init.SetRef(stage.Ref() + ".v")
			init.AddCondition(stage.Condition(), stage.Ref()+".v")
			init.PrependInitialStatus(stage.Ref() + ".v")
			init.AddResult("!"+stage.Ref()+".v", stage.Ref())
		} else if stage.Ref() != "" {
			init.PrependInitialStatus(stage.Ref())
		}

		if group.Optional() {
			init.ResetResults()
		}

		if group.Negative() {
			init.AddResult(strings.Join(directRefResults, "&&"), stage.Ref()+".v")
		} else {
			init.AddResult(strings.Join(directRefResults, "&&"), stage.Ref())
		}

		for i, ch := range group.Children() {
			// Condition should be executed only in the first leaf
			if i == 1 {
				init.ResetCondition().SetPaused(false)
			}
			// Pass down to another group or container
			sub, serr := buildKubernetesContainers(ch, init.Children(ch.Ref()), fsGroup, machines...)
			if serr != nil {
				return nil, fmt.Errorf("%s: %s: resolving children: %s", stage.Ref(), stage.Name(), serr.Error())
			}
			containers = append(containers, sub...)
		}
		return
	}
	c, ok := stage.(ContainerStage)
	if !ok {
		return nil, fmt.Errorf("%s: %s: stage that is neither container nor group", stage.Ref(), stage.Name())
	}
	err = c.Container().Detach().Resolve(machines...)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: resolving container: %s", stage.Ref(), stage.Name(), err.Error())
	}

	cr, err := c.Container().ToKubernetesTemplate()
	if err != nil {
		return nil, fmt.Errorf("%s: %s: building container template: %s", stage.Ref(), stage.Name(), err.Error())
	}
	cr.Name = c.Ref()

	if c.Optional() {
		init.ResetResults()
	}

	bypass := false
	refEnvVar := ""
	for _, e := range cr.Env {
		if e.Name == BypassToolkitCheck.Name && e.Value == BypassToolkitCheck.Value {
			bypass = true
		}
		if e.Name == "TK_REF" {
			refEnvVar = e.Value
		}
	}

	init.
		SetNegative(c.Negative()).
		AddRetryPolicy(c.RetryPolicy(), c.Ref()).
		SetCommand(cr.Command...).
		SetArgs(cr.Args...).
		SetWorkingDir(cr.WorkingDir).
		SetToolkit(bypass || (cr.Image == constants.DefaultToolkitImage && c.Ref() == refEnvVar))

	for _, env := range cr.Env {
		if strings.Contains(env.Value, "{{") {
			init.AddComputedEnvs(env.Name)
		}
	}

	if init.Error() != nil {
		return nil, init.Error()
	}

	cr.Command = init.Command()
	cr.Args = init.Args()
	cr.WorkingDir = ""

	// Ensure the container will have proper access to FS
	if cr.SecurityContext == nil {
		cr.SecurityContext = &corev1.SecurityContext{}
	}
	if cr.SecurityContext.RunAsGroup == nil {
		cr.SecurityContext.RunAsGroup = fsGroup
	}

	containers = []corev1.Container{cr}
	return
}
