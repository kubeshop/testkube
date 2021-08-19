package scripts

import (
	scriptsV1 "github.com/kubeshop/kubtest-operator/apis/script/v1"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

func MapScriptListKubeToAPI(crScripts scriptsV1.ScriptList) (scripts []kubtest.Script) {
	for _, item := range crScripts.Items {
		scripts = append(scripts, MapScriptKubeToAPI(item))
	}

	return
}
func MapScriptKubeToAPI(crScript scriptsV1.Script) (script kubtest.Script) {
	script.Name = crScript.Name
	script.Content = crScript.Spec.Content
	script.Created = crScript.Status.LastExecution.Time
	script.Type_ = crScript.Spec.Type

	return
}
