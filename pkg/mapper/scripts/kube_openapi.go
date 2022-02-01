package scripts

import (
	scriptsv2 "github.com/kubeshop/testkube-operator/apis/script/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapScriptListKubeToAPI(crScripts scriptsv2.ScriptList) (scripts []testkube.Script) {
	for _, item := range crScripts.Items {
		scripts = append(scripts, MapScriptCRToAPI(item))
	}

	return
}
func MapScriptCRToAPI(crScript scriptsv2.Script) (script testkube.Script) {
	script.Name = crScript.Name
	script.Content = MapScriptContentFromSpec(crScript.Spec.Content)
	script.Created = crScript.CreationTimestamp.Time
	script.Type_ = crScript.Spec.Type_
	script.Tags = crScript.Spec.Tags
	return
}

func MapScriptContentFromSpec(specContent *scriptsv2.ScriptContent) *testkube.ScriptContent {
	content := &testkube.ScriptContent{
		Type_: specContent.Type_,
		// assuming same data structure - there is task about syncing them automatically
		// https://github.com/kubeshop/testkube/issues/723
		Repository: (*testkube.Repository)(specContent.Repository),
		Data:       specContent.Data,
		Uri:        specContent.Uri,
	}

	return content
}
