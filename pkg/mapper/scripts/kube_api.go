package scripts

import (
	scriptsV1 "github.com/kubeshop/testkube-operator/apis/script/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapScriptListKubeToAPI(crScripts scriptsV1.ScriptList) (scripts []testkube.Script) {
	for _, item := range crScripts.Items {
		scripts = append(scripts, MapScriptCRToAPI(item))
	}

	return
}
func MapScriptCRToAPI(crScript scriptsV1.Script) (script testkube.Script) {
	script.Name = crScript.Name
	script.Content = crScript.Spec.Content
	script.Created = crScript.Status.LastExecution.Time
	script.Type_ = crScript.Spec.Type_

	return
}
