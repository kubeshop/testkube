package testkube

type ScriptContentType string

const (
	ScriptContentTypeString  ScriptContentType = "string"
	ScriptContentTypeFileURI ScriptContentType = "file-uri"
	ScriptContentTypeGitFile ScriptContentType = "git-file"
	ScriptContentTypeGitDir  ScriptContentType = "git-dir"
)

func NewStringScriptContent(str string) *ScriptContent {
	return &ScriptContent{
		Type_: string(ScriptContentTypeGitFile),
		Data:  str,
	}
}
