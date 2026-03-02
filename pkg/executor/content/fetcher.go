package content

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/git"
	"github.com/kubeshop/testkube/pkg/ui"
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

// FetchGit returns path to git based file or dir saved in local temp directory
func (f Fetcher) FetchGit(repo *testkube.Repository) (path string, err error) {
	uri, authHeader, err := f.gitURI(repo)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to fetch git: %s", ui.IconCross, err.Error()))
		return path, err
	}

	// if path not set make full repo checkout
	if repo.Path == "" || repo.WorkingDir != "" {
		path, err := git.Checkout(uri, authHeader, repo.Branch, repo.Commit, f.path)
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Failed to fetch git: %s", ui.IconCross, err.Error()))
			return path, fmt.Errorf("failed to fetch git: %w", err)
		}

		if repo.Path != "" {
			fileInfo, err := os.Stat(filepath.Join(path, repo.Path))
			if err != nil {
				output.PrintLog(fmt.Sprintf("%s Failed to get file stat: %s", ui.IconCross, err.Error()))
				return path, fmt.Errorf("failed to get file stat: %w", err)
			}

			if !fileInfo.IsDir() {
				path = filepath.Join(path, repo.Path)
			}
		}

		output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
		return path, nil
	}

	path, err = git.PartialCheckout(uri, authHeader, repo.Path, repo.Branch, repo.Commit, f.path)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to do partial checkout on git: %s", ui.IconCross, err.Error()))
		return path, fmt.Errorf("failed to do partial checkout on git: %w", err)
	}

	output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
	return path, nil
}

// gitUri merge creds with git uri
func (f Fetcher) gitURI(repo *testkube.Repository) (uri, authHeader string, err error) {
	if repo.AuthType == string(testkube.GitAuthTypeHeader) {
		if repo.Token != "" {
			authHeader = fmt.Sprintf("Authorization: Bearer %s", repo.Token)
			return repo.Uri, authHeader, nil
		}
	}

	if repo.AuthType == string(testkube.GitAuthTypeBasic) || repo.AuthType == "" {
		if repo.Username != "" || repo.Token != "" {
			gitURI, err := url.Parse(repo.Uri)
			if err != nil {
				return "", "", err
			}

			gitURI.User = url.UserPassword(repo.Username, repo.Token)
			return gitURI.String(), "", nil
		}
	}

	return repo.Uri, "", nil
}
