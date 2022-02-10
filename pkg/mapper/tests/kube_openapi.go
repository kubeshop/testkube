package tests

import (
	scriptsv2 "github.com/kubeshop/testkube-operator/apis/script/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapScriptListKubeToAPI(crScripts scriptsv2.ScriptList) (scripts []testkube.Test) {
	for _, item := range crScripts.Items {
		scripts = append(scripts, MapScriptCRToAPI(item))
	}

	return
}
func MapScriptCRToAPI(crScript scriptsv2.Script) (test testkube.Test) {
	test.Name = crScript.Name
	test.Content = MapScriptContentFromSpec(crScript.Spec.Content)
	test.Created = crScript.CreationTimestamp.Time
	test.Type_ = crScript.Spec.Type_
	test.Tags = crScript.Spec.Tags
	return
}

func MapScriptContentFromSpec(specContent *scriptsv2.ScriptContent) *testkube.TestContent {
	content := &testkube.TestContent{
		Type_: specContent.Type_,
		// assuming same data structure - there is task about syncing them automatically
		// https://github.com/kubeshop/testkube/issues/723
		Repository: (*testkube.Repository)(specContent.Repository),
		Data:       specContent.Data,
		Uri:        specContent.Uri,
	}

	return content
}
