package mock

import (
	"log"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Fetcher implements the Mock version of the content fetcher from 	"github.com/kubeshop/testkube/pkg/executor/content"
type Fetcher struct {
	FetchFn                     func(content *testkube.TestContent) (path string, err error)
	FetchStringFn               func(str string) (path string, err error)
	FetchURIFn                  func(uri string) (path string, err error)
	FetchGitDirFn               func(repo *testkube.Repository) (path string, err error)
	FetchGitFileFn              func(repo *testkube.Repository) (path string, err error)
	FetchGitFn                  func(repo *testkube.Repository) (path string, err error)
	FetchCalculateContentTypeFn func(repo testkube.Repository) (string, error)
}

func (f Fetcher) Fetch(content *testkube.TestContent) (path string, err error) {
	if f.FetchFn == nil {
		log.Fatal("not implemented")
	}
	return f.FetchFn(content)
}

func (f Fetcher) FetchString(str string) (path string, err error) {
	if f.FetchStringFn == nil {
		log.Fatal("not implemented")
	}
	return f.FetchStringFn(str)
}

func (f Fetcher) FetchURI(str string) (path string, err error) {
	if f.FetchURIFn == nil {
		log.Fatal("not implemented")
	}
	return f.FetchURIFn(str)
}

func (f Fetcher) FetchGitDir(repo *testkube.Repository) (path string, err error) {
	if f.FetchGitDirFn == nil {
		log.Fatal("not implemented")
	}
	return f.FetchGitDir(repo)
}

func (f Fetcher) FetchGitFile(repo *testkube.Repository) (path string, err error) {
	if f.FetchGitFileFn == nil {
		log.Fatal("not implemented")
	}
	return f.FetchGitFileFn(repo)
}

func (f Fetcher) FetchGit(repo *testkube.Repository) (path string, err error) {
	if f.FetchGitFn == nil {
		log.Fatal("not implemented")
	}
	return f.FetchGitFn(repo)
}

func (f Fetcher) CalculateGitContentType(repo testkube.Repository) (string, error) {
	if f.FetchCalculateContentTypeFn == nil {
		log.Fatal("not implemented")
	}
	return f.FetchCalculateContentTypeFn(repo)
}
