package workflowtriggers

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

// MapCRDToAPI converts the Kubernetes CRD shape (spec-wrapped) into the flat
// REST API shape consumed by CLI and control plane.
func MapCRDToAPI(crd *workflowtriggersv1.WorkflowTrigger) testkube.WorkflowTrigger {
	if crd == nil {
		return testkube.WorkflowTrigger{}
	}
	return testkube.WorkflowTrigger{
		Name:        crd.Name,
		Namespace:   crd.Namespace,
		Labels:      crd.Labels,
		Annotations: crd.Annotations,
		Disabled:    crd.Spec.Disabled,
		Watch:       mapWatchCRDToAPI(crd.Spec.Watch),
		When:        testkube.WorkflowTriggerWhen{Event: crd.Spec.When.Event},
		Match:       mapMatchCRDToAPI(crd.Spec.Match),
		Wait:        mapWaitCRDToAPI(crd.Spec.Wait),
		Run:         mapRunCRDToAPI(crd.Spec.Run),
	}
}

// MapAPIToCRD converts the flat REST shape into the CRD shape for K8s persistence.
func MapAPIToCRD(api testkube.WorkflowTrigger) workflowtriggersv1.WorkflowTrigger {
	return workflowtriggersv1.WorkflowTrigger{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflowtriggersv1.Kind,
			APIVersion: workflowtriggersv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        api.Name,
			Namespace:   api.Namespace,
			Labels:      api.Labels,
			Annotations: api.Annotations,
		},
		Spec: workflowtriggersv1.WorkflowTriggerSpec{
			Disabled: api.Disabled,
			Watch:    mapWatchAPIToCRD(api.Watch),
			When:     workflowtriggersv1.WorkflowTriggerWhen{Event: api.When.Event},
			Match:    mapMatchAPIToCRD(api.Match),
			Wait:     mapWaitAPIToCRD(api.Wait),
			Run:      mapRunAPIToCRD(api.Run),
		},
	}
}

// MapListCRDToAPI maps a CRD list into a flat API slice.
func MapListCRDToAPI(list *workflowtriggersv1.WorkflowTriggerList) []testkube.WorkflowTrigger {
	if list == nil {
		return nil
	}
	out := make([]testkube.WorkflowTrigger, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, MapCRDToAPI(&list.Items[i]))
	}
	return out
}

func mapWatchCRDToAPI(w *workflowtriggersv1.WorkflowTriggerWatch) *testkube.WorkflowTriggerWatch {
	if w == nil {
		return nil
	}
	return &testkube.WorkflowTriggerWatch{
		Resource: testkube.WorkflowTriggerResource{
			Group:     w.Resource.Group,
			Version:   w.Resource.Version,
			Kind:      w.Resource.Kind,
			Name:      w.Resource.Name,
			Namespace: w.Resource.Namespace,
		},
		Selector: mapSelectorCRDToAPI(w.Selector),
	}
}

func mapWatchAPIToCRD(w *testkube.WorkflowTriggerWatch) *workflowtriggersv1.WorkflowTriggerWatch {
	if w == nil {
		return nil
	}
	return &workflowtriggersv1.WorkflowTriggerWatch{
		Resource: workflowtriggersv1.WorkflowTriggerResource{
			Group:     w.Resource.Group,
			Version:   w.Resource.Version,
			Kind:      w.Resource.Kind,
			Name:      w.Resource.Name,
			Namespace: w.Resource.Namespace,
		},
		Selector: mapSelectorAPIToCRD(w.Selector),
	}
}

func mapSelectorCRDToAPI(s *workflowtriggersv1.WorkflowTriggerSelector) *testkube.WorkflowTriggerSelector {
	if s == nil {
		return nil
	}
	return &testkube.WorkflowTriggerSelector{
		NameRegex:      s.NameRegex,
		NamespaceRegex: s.NamespaceRegex,
		LabelSelector:  mapLabelSelectorCRDToAPI(s.LabelSelector),
	}
}

func mapSelectorAPIToCRD(s *testkube.WorkflowTriggerSelector) *workflowtriggersv1.WorkflowTriggerSelector {
	if s == nil {
		return nil
	}
	return &workflowtriggersv1.WorkflowTriggerSelector{
		NameRegex:      s.NameRegex,
		NamespaceRegex: s.NamespaceRegex,
		LabelSelector:  mapLabelSelectorAPIToCRD(s.LabelSelector),
	}
}

func mapLabelSelectorCRDToAPI(l *metav1.LabelSelector) *testkube.WorkflowTriggerLabelSel {
	if l == nil {
		return nil
	}
	return &testkube.WorkflowTriggerLabelSel{MatchLabels: l.MatchLabels}
}

func mapLabelSelectorAPIToCRD(l *testkube.WorkflowTriggerLabelSel) *metav1.LabelSelector {
	if l == nil {
		return nil
	}
	return &metav1.LabelSelector{MatchLabels: l.MatchLabels}
}

func mapMatchCRDToAPI(items []workflowtriggersv1.WorkflowTriggerFieldCondition) []testkube.WorkflowTriggerFieldCondition {
	if len(items) == 0 {
		return nil
	}
	out := make([]testkube.WorkflowTriggerFieldCondition, 0, len(items))
	for _, m := range items {
		out = append(out, testkube.WorkflowTriggerFieldCondition{
			Path:     m.Path,
			Operator: string(m.Operator),
			Value:    m.Value,
		})
	}
	return out
}

func mapMatchAPIToCRD(items []testkube.WorkflowTriggerFieldCondition) []workflowtriggersv1.WorkflowTriggerFieldCondition {
	if len(items) == 0 {
		return nil
	}
	out := make([]workflowtriggersv1.WorkflowTriggerFieldCondition, 0, len(items))
	for _, m := range items {
		out = append(out, workflowtriggersv1.WorkflowTriggerFieldCondition{
			Path:     m.Path,
			Operator: workflowtriggersv1.WorkflowTriggerFieldOperator(m.Operator),
			Value:    m.Value,
		})
	}
	return out
}

func mapWaitCRDToAPI(w *workflowtriggersv1.WorkflowTriggerWait) *testkube.WorkflowTriggerWait {
	if w == nil {
		return nil
	}
	out := &testkube.WorkflowTriggerWait{}
	if w.Conditions != nil {
		out.Conditions = &testkube.WorkflowTriggerWaitConditions{
			Timeout: w.Conditions.Timeout,
			Delay:   w.Conditions.Delay,
		}
		for _, c := range w.Conditions.Items {
			status := ""
			if c.Status != nil {
				status = string(*c.Status)
			}
			out.Conditions.Items = append(out.Conditions.Items, testkube.WorkflowTriggerCondition{
				Type: c.Type, Status: status, Reason: c.Reason, TTL: c.TTL,
			})
		}
	}
	if w.Probes != nil {
		out.Probes = &testkube.WorkflowTriggerWaitProbes{
			Timeout: w.Probes.Timeout,
			Delay:   w.Probes.Delay,
		}
		for _, p := range w.Probes.Items {
			out.Probes.Items = append(out.Probes.Items, testkube.WorkflowTriggerProbe{
				Scheme: p.Scheme, Host: p.Host, Path: p.Path, Port: p.Port, Headers: p.Headers,
			})
		}
	}
	return out
}

func mapWaitAPIToCRD(w *testkube.WorkflowTriggerWait) *workflowtriggersv1.WorkflowTriggerWait {
	if w == nil {
		return nil
	}
	out := &workflowtriggersv1.WorkflowTriggerWait{}
	if w.Conditions != nil {
		out.Conditions = &workflowtriggersv1.WorkflowTriggerWaitConditions{
			Timeout: w.Conditions.Timeout,
			Delay:   w.Conditions.Delay,
		}
		for _, c := range w.Conditions.Items {
			status := workflowtriggersv1.WorkflowTriggerConditionStatus(c.Status)
			out.Conditions.Items = append(out.Conditions.Items, workflowtriggersv1.WorkflowTriggerCondition{
				Type: c.Type, Status: &status, Reason: c.Reason, TTL: c.TTL,
			})
		}
	}
	if w.Probes != nil {
		out.Probes = &workflowtriggersv1.WorkflowTriggerWaitProbes{
			Timeout: w.Probes.Timeout,
			Delay:   w.Probes.Delay,
		}
		for _, p := range w.Probes.Items {
			out.Probes.Items = append(out.Probes.Items, workflowtriggersv1.WorkflowTriggerProbe{
				Scheme: p.Scheme, Host: p.Host, Path: p.Path, Port: p.Port, Headers: p.Headers,
			})
		}
	}
	return out
}

func mapRunCRDToAPI(r workflowtriggersv1.WorkflowTriggerRun) testkube.WorkflowTriggerRun {
	out := testkube.WorkflowTriggerRun{
		Workflow: testkube.WorkflowTriggerWorkflowSelector{
			Name:          r.Workflow.Name,
			NameRegex:     r.Workflow.NameRegex,
			LabelSelector: mapLabelSelectorCRDToAPI(r.Workflow.LabelSelector),
		},
		ConcurrencyPolicy: r.ConcurrencyPolicy,
	}
	if r.Parameters != nil {
		out.Parameters = &testkube.WorkflowTriggerRunParameters{
			Config: r.Parameters.Config,
			Tags:   r.Parameters.Tags,
		}
	}
	if r.Delay != nil {
		out.Delay = r.Delay.Duration.String()
	}
	return out
}

func mapRunAPIToCRD(r testkube.WorkflowTriggerRun) workflowtriggersv1.WorkflowTriggerRun {
	out := workflowtriggersv1.WorkflowTriggerRun{
		Workflow: workflowtriggersv1.WorkflowTriggerWorkflowSelector{
			Name:          r.Workflow.Name,
			NameRegex:     r.Workflow.NameRegex,
			LabelSelector: mapLabelSelectorAPIToCRD(r.Workflow.LabelSelector),
		},
		ConcurrencyPolicy: r.ConcurrencyPolicy,
	}
	if r.Parameters != nil {
		out.Parameters = &workflowtriggersv1.WorkflowTriggerRunParameters{
			Config: r.Parameters.Config,
			Tags:   r.Parameters.Tags,
		}
	}
	if r.Delay != "" {
		d, err := time.ParseDuration(r.Delay)
		if err != nil {
			// Caller should validate at request boundary; drop the field and log so
			// the CRD is still well-formed. Never write to stdout from a library.
			log.DefaultLogger.Warnw("workflowtriggers mapper: invalid delay, dropping", "delay", r.Delay, "error", err)
		} else {
			out.Delay = &metav1.Duration{Duration: d}
		}
	}
	return out
}
