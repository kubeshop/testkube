package scripts

import (
	scriptsV1 "github.com/kubeshop/kubtest-operator/apis/script/v1"
	"github.com/kubeshop/kubtest/pkg/api/v1/kubtest"
)

func MapScriptListKubeToAPI(crScripts scriptsV1.ScriptList) (scripts []kubtest.Script) {
	for _, item := range crScripts.Items {
		scripts = append(scripts, MapScriptCRToAPI(item))
	}

	return
}
func MapScriptCRToAPI(crScript scriptsV1.Script) (script kubtest.Script) {
	script.Name = crScript.Name
	script.Content = crScript.Spec.Content
	script.Created = crScript.Status.LastExecution.Time
	script.Type_ = crScript.Spec.Type_

	return
}
