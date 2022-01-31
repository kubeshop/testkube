package content

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/git"
)

func NewFetcher() Fetcher {
	return Fetcher{}
}

type Fetcher struct {
}

func (f Fetcher) Fetch(content testkube.ScriptContent) (path string, err error) {
	switch testkube.ScriptContentType(content.Type_) {
	case testkube.ScriptContentTypeFileURI:
		return f.FetchURI(content.Uri)
	case testkube.ScriptContentTypeString:
		return f.FetchString(content.Data)
	case testkube.ScriptContentTypeGitFile:
		return f.FetchGitFile(content.Repository)
	case testkube.ScriptContentTypeGitDir:
		return f.FetchGitDir(content.Repository)
	default:
		return path, fmt.Errorf("can't fetch %s", content.Type_)
	}
}

// FetchString stores string content as file
func (f Fetcher) FetchString(str string) (path string, err error) {
	return f.saveTempFile(strings.NewReader(str))
}

//FetchURI stores uri as local file
func (f Fetcher) FetchURI(uri string) (path string, err error) {
	resp, err := http.Get(uri)
	if err != nil {
		return path, err
	}
	defer resp.Body.Close()

	return f.saveTempFile(resp.Body)
}

// FetchGitDir returns path to locally checked out git repo with partial path
func (f Fetcher) FetchGitDir(repo *testkube.Repository) (path string, err error) {
	uri, err := f.gitURI(repo)
	if err != nil {
		return path, err
	}

	// if . make full repo checkout
	if repo.Path == "." {
		return git.Checkout(uri, repo.Branch)
	}

	return git.PartialCheckout(uri, repo.Path, repo.Branch)
}

// FetchGitFile returns path to git based file saved in local temp directory
func (f Fetcher) FetchGitFile(repo *testkube.Repository) (path string, err error) {
	uri, err := f.gitURI(repo)
	if err != nil {
		return path, err
	}

	repoPath, err := git.Checkout(uri, repo.Branch)
	if err != nil {
		return path, err
	}

	// get git file
	return filepath.Join(repoPath, repo.Path), nil
}

// gitUri merge creds with git uri
func (f Fetcher) gitURI(repo *testkube.Repository) (uri string, err error) {
	if repo.Username != "" && repo.Token != "" {
		gitURI, err := url.Parse(repo.Uri)
		if err != nil {
			return uri, err
		}

		gitURI.User = url.UserPassword(repo.Username, repo.Token)
		return gitURI.String(), nil
	}

	return repo.Uri, nil
}

func (f Fetcher) saveTempFile(reader io.Reader) (path string, err error) {
	tmpFile, err := ioutil.TempFile("", "fetcher-save-temp-file")
	path = tmpFile.Name()
	out, err := os.Create(path)
	if err != nil {
		return path, err
	}
	defer out.Close()
	_, err = io.Copy(out, reader)

	return
}
