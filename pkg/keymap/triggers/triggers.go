package triggers

import "github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"

type KeyMap struct {
	Resources           []string            `json:"resources"`
	Actions             []string            `json:"actions"`
	Executions          []string            `json:"executions"`
	Events              map[string][]string `json:"events"`
	Conditions          []string            `json:"conditions"`
	ConcurrencyPolicies []string            `json:"concurrencyPolicies"`
}

func NewKeyMap() *KeyMap {
	return &KeyMap{
		Resources:           testtrigger.GetSupportedResources(),
		Actions:             testtrigger.GetSupportedActions(),
		Executions:          testtrigger.GetSupportedExecutions(),
		Events:              getSupportedEvents(),
		Conditions:          testtrigger.GetSupportedConditions(),
		ConcurrencyPolicies: testtrigger.GetSupportedConcurrencyPolicies(),
	}
}

func getSupportedEvents() map[string][]string {
	m := make(map[string][]string, len(testtrigger.GetSupportedResources()))
	m[testtrigger.ResourcePod] = []string{string(testtrigger.EventCreated), string(testtrigger.EventModified), string(testtrigger.EventDeleted)}
	m[testtrigger.ResourceDeployment] = []string{
		string(testtrigger.EventCreated),
		string(testtrigger.EventModified),
		string(testtrigger.EventDeleted),
		string(testtrigger.CauseDeploymentContainersModified),
		string(testtrigger.CauseDeploymentImageUpdate),
		string(testtrigger.CauseDeploymentScaleUpdate),
		string(testtrigger.CauseDeploymentEnvUpdate),
	}
	m[testtrigger.ResourceStatefulSet] = []string{string(testtrigger.EventCreated), string(testtrigger.EventModified), string(testtrigger.EventDeleted)}
	m[testtrigger.ResourceDaemonSet] = []string{string(testtrigger.EventCreated), string(testtrigger.EventModified), string(testtrigger.EventDeleted)}
	m[testtrigger.ResourceService] = []string{string(testtrigger.EventCreated), string(testtrigger.EventModified), string(testtrigger.EventDeleted)}
	m[testtrigger.ResourceIngress] = []string{string(testtrigger.EventCreated), string(testtrigger.EventModified), string(testtrigger.EventDeleted)}
	m[testtrigger.ResourceEvent] = []string{string(testtrigger.EventCreated), string(testtrigger.EventModified), string(testtrigger.EventDeleted)}
	m[testtrigger.ResourceConfigMap] = []string{string(testtrigger.EventCreated), string(testtrigger.EventModified), string(testtrigger.EventDeleted)}
	return m
}
