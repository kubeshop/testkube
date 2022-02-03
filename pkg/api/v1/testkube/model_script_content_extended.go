// content could be fetched as file or dir (many files, e.g. Cypress project) in executor
package testkube

import "fmt"

type ScriptContentType string

const (
	ScriptContentTypeString  ScriptContentType = "string"
	ScriptContentTypeFileURI ScriptContentType = "file-uri"
	ScriptContentTypeGitFile ScriptContentType = "git-file"
	ScriptContentTypeGitDir  ScriptContentType = "git-dir"
)

var ErrScriptContentTypeNotFile = fmt.Errorf("unsupported content type use one of: file-uri, git-file, string")
var ErrScriptContentTypeNotDir = fmt.Errorf("unsupported content type use one of: git-dir")

func NewStringScriptContent(str string) *ScriptContent {
	return &ScriptContent{
		Type_: string(ScriptContentTypeGitFile),
		Data:  str,
	}
}

// IsDir - for content fetched as dir
func (c *ScriptContent) IsDir() bool {
	return ScriptContentType(c.Type_) == ScriptContentTypeGitDir

}

// IsFile - for content fetched as file
func (c *ScriptContent) IsFile() bool {
	return ScriptContentType(c.Type_) == ScriptContentTypeGitFile ||
		ScriptContentType(c.Type_) == ScriptContentTypeFileURI ||
		ScriptContentType(c.Type_) == ScriptContentTypeString
}
