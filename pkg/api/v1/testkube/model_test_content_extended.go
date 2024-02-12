// content could be fetched as file or dir (many files, e.g. Cypress project) in executor
package testkube

import "fmt"

type TestContentType string

const (
	TestContentTypeString  TestContentType = "string"
	TestContentTypeFileURI TestContentType = "file-uri"
	TestContentTypeGitFile TestContentType = "git-file"
	TestContentTypeGitDir  TestContentType = "git-dir"
	TestContentTypeGit     TestContentType = "git"
	TestContentTypeEmpty   TestContentType = ""
)

type ArgsModeType string

const (
	ArgsModeTypeAppend   ArgsModeType = "append"
	ArgsModeTypeOverride ArgsModeType = "override"
	ArgsModeTypeReplace  ArgsModeType = "replace"
)

var ErrTestContentTypeNotFile = fmt.Errorf("unsupported content type use one of: file-uri, git-file, string")
var ErrTestContentTypeNotDir = fmt.Errorf("unsupported content type use one of: git-dir")

func NewStringTestContent(str string) *TestContent {
	return &TestContent{
		Type_: string(TestContentTypeString),
		Data:  str,
	}
}

// IsDir - for content fetched as dir
//
// Deprecated: check source data
func (c *TestContent) IsDir() bool {
	return TestContentType(c.Type_) == TestContentTypeGitDir

}

// IsFile - for content fetched as file
// Deprected: check source data
func (c *TestContent) IsFile() bool {
	return TestContentType(c.Type_) == TestContentTypeGitFile ||
		TestContentType(c.Type_) == TestContentTypeFileURI ||
		TestContentType(c.Type_) == TestContentTypeString
}
