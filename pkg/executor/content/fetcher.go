package content

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/git"
	"github.com/kubeshop/testkube/pkg/http"
)

// NewFetcher returns new file/dir fetcher based on given directory path
func NewFetcher(path string) Fetcher {
	return Fetcher{
		path: path,
	}
}

type Fetcher struct {
	path string
}

func (f Fetcher) Fetch(content *testkube.TestContent) (path string, err error) {
	if content == nil {
		return "", fmt.Errorf("fetch - empty content, make sure test content has valid data structure and is not nil")
	}
	switch testkube.TestContentType(content.Type_) {
	case testkube.TestContentTypeFileURI:
		return f.FetchURI(content.Uri)
	case testkube.TestContentTypeString:
		return f.FetchString(content.Data)
	case testkube.TestContentTypeGitFile:
		return f.FetchGitFile(content.Repository)
	case testkube.TestContentTypeGitDir:
		return f.FetchGitDir(content.Repository)
	default:
		return path, fmt.Errorf("unhandled content type: '%s'", content.Type_)
	}
}

// FetchString stores string content as file
func (f Fetcher) FetchString(str string) (path string, err error) {
	return f.saveTempFile(strings.NewReader(str))
}

//FetchURI stores uri as local file
func (f Fetcher) FetchURI(uri string) (path string, err error) {
	client := http.NewClient()
	resp, err := client.Get(uri)
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

	// if path not set make full repo checkout
	if repo.Path == "" {
		return git.Checkout(uri, repo.Branch, repo.Commit, f.path)
	}

	return git.PartialCheckout(uri, repo.Path, repo.Branch, repo.Commit, f.path)
}

// FetchGitFile returns path to git based file saved in local temp directory
func (f Fetcher) FetchGitFile(repo *testkube.Repository) (path string, err error) {
	uri, err := f.gitURI(repo)
	if err != nil {
		return path, err
	}

	repoPath, err := git.Checkout(uri, repo.Branch, repo.Commit, f.path)
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
	var tmpFile *os.File
	filename := "test-content"
	if f.path == "" {
		tmpFile, err = os.CreateTemp("", filename)
	} else {
		tmpFile, err = os.Create(filepath.Join(f.path, filename))
	}
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	_, err = io.Copy(tmpFile, reader)

	return tmpFile.Name(), err
}
