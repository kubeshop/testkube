package scripts

import (
	scriptsV1 "github.com/kubeshop/kubetest/internal/app/operator/api/v1"
	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

func MapScriptListKubeToAPI(crScripts scriptsV1.ScriptList) (scripts []kubetest.Script) {
	for _, item := range crScripts.Items {
		scripts = append(scripts, MapScriptKubeToAPI(item))
	}

	return
}
func MapScriptKubeToAPI(crScript scriptsV1.Script) (script kubetest.Script) {
	script.Name = crScript.Name
	script.Content = crScript.Spec.Content
	script.Created = crScript.Status.LastExecution.Time
	script.Type_ = crScript.Spec.Type

	return
}
