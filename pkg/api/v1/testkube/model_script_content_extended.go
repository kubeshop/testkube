package testkube

func NewStringScriptContent(str string) *ScriptContent {
	return &ScriptContent{
		Type_: "string",
		Data:  str,
	}
}
