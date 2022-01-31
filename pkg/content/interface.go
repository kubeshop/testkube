package content

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type ContentFetcher interface {
	StringFetcher
	URIFetcher
	GitDirFetcher
	GitFileFetcher

	Fetch(content testkube.ScriptContent) (path string, err error)
}

type StringFetcher interface {
	FetchString(str string) (path string, err error)
}

type URIFetcher interface {
	FetchURI(uri string) (path string, err error)
}

type GitDirFetcher interface {
	FetchGitDir(repo testkube.Repository) (path string, err error)
}

type GitFileFetcher interface {
	FetchGitFile(repo testkube.Repository) (path string, err error)
}
