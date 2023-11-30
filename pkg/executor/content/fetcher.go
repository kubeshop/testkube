package content

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"net/url"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/git"
	"github.com/kubeshop/testkube/pkg/ui"

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
		output.PrintLog(fmt.Sprintf("%s Fetch - empty content, make sure test content has valid data structure and is not nil", ui.IconCross))
		return "", fmt.Errorf("fetch - empty content, make sure test content has valid data structure and is not nil")
	}

	if content.Repository != nil && content.Repository.CertificateSecret != "" {
		output.PrintLog(fmt.Sprintf("%s Using certificate for git authentication from secret: %s", ui.IconBox, content.Repository.CertificateSecret))
		f.configureUseOfCertificate()
	}

	output.PrintLog(fmt.Sprintf("%s Fetching test content from %s...", ui.IconBox, content.Type_))

	switch testkube.TestContentType(content.Type_) {
	case testkube.TestContentTypeFileURI:
		return f.FetchURI(content.Uri)
	case testkube.TestContentTypeString:
		return f.FetchString(content.Data)
	case testkube.TestContentTypeGitFile:
		return f.FetchGitFile(content.Repository)
	case testkube.TestContentTypeGitDir:
		return f.FetchGitDir(content.Repository)
	case testkube.TestContentTypeGit:
		return f.FetchGit(content.Repository)
	case testkube.TestContentTypeEmpty:
		output.PrintLog(fmt.Sprintf("%s Empty content type", ui.IconCross))
		return path, nil
	default:
		output.PrintLog(fmt.Sprintf("%s Unhandled content type: '%s'", ui.IconCross, content.Type_))
		return path, fmt.Errorf("unhandled content type: '%s'", content.Type_)
	}
}

// FetchString stores string content as file
func (f Fetcher) FetchString(str string) (path string, err error) {
	return f.saveTempFile(strings.NewReader(str))
}

// FetchURI stores uri as local file
func (f Fetcher) FetchURI(uri string) (path string, err error) {
	client := http.NewClient()
	resp, err := client.Get(uri)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to fetch test content: %s", ui.IconCross, err.Error()))
		return path, err
	}
	defer resp.Body.Close()

	return f.saveTempFile(resp.Body)
}

// FetchGitDir returns path to locally checked out git repo with partial path
func (f Fetcher) FetchGitDir(repo *testkube.Repository) (path string, err error) {
	uri, authHeader, err := f.gitURI(repo)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to fetch git dir: %s", ui.IconCross, err.Error()))
		return path, err
	}

	// if path not set make full repo checkout
	if repo.Path == "" || repo.WorkingDir != "" {
		path, err := git.Checkout(uri, authHeader, repo.Branch, repo.Commit, f.path)
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Failed to fetch git dir: %s", ui.IconCross, err.Error()))
			return path, fmt.Errorf("failed to fetch git dir: %w", err)
		}
		output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
		return path, nil
	}

	path, err = git.PartialCheckout(uri, authHeader, repo.Path, repo.Branch, repo.Commit, f.path)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to do partial checkout on git dir: %s", ui.IconCross, err.Error()))
		return path, fmt.Errorf("failed to do partial checkout on git dir: %w", err)
	}
	output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
	return path, nil
}

// FetchGitFile returns path to git based file saved in local temp directory
func (f Fetcher) FetchGitFile(repo *testkube.Repository) (path string, err error) {
	uri, authHeader, err := f.gitURI(repo)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to fetch git file: %s", ui.IconCross, err.Error()))
		return path, err
	}

	repoPath, err := git.Checkout(uri, authHeader, repo.Branch, repo.Commit, f.path)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to checkout git file: %s", ui.IconCross, err.Error()))
		return path, err
	}

	path = filepath.Join(repoPath, repo.Path)
	output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
	return path, nil
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

func (f Fetcher) saveTempFile(reader io.Reader) (path string, err error) {
	var tmpFile *os.File
	filename := "test-content"
	if f.path == "" {
		tmpFile, err = os.CreateTemp("", filename)
	} else {
		tmpFile, err = os.Create(filepath.Join(f.path, filename))
	}
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to save test content: %s", ui.IconCross, err.Error()))
		return "", err
	}
	defer tmpFile.Close()
	_, err = io.Copy(tmpFile, reader)

	output.PrintLog(fmt.Sprintf("%s Content saved to path %s", ui.IconCheckMark, tmpFile.Name()))
	return tmpFile.Name(), err
}

func (f Fetcher) configureUseOfCertificate() error {
	const certsPath = "/etc/certs"
	const certExtension = ".crt"
	var certificatePath string
	output.PrintLog(fmt.Sprintf("%s Fetching certificate from path %s", ui.IconCheckMark, certsPath))
	err := filepath.Walk(certsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Failed to walk through %s to find certificates: %s", ui.IconCross, certsPath, err.Error()))
			return err
		}
		if filepath.Ext(path) == certExtension {
			output.PrintLog(fmt.Sprintf("%s Found certificate %s", ui.IconCheckMark, path))
			certificatePath = path
		}
		return nil
	})
	if err != nil || certificatePath == "" {
		output.PrintLog(fmt.Sprintf("%s Failed to find certificate in %s: %s", ui.IconCross, certsPath, err.Error()))
		return err
	}
	gitConfigCommand := fmt.Sprintf("git config --global http.sslCAInfo %s", certificatePath)
	out, err := exec.Command("/bin/sh", "-c", gitConfigCommand).Output()
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to configure git to use certificate %s, output:%s \n Error: %s", ui.IconCross, certificatePath, out, err.Error()))
		return err
	}
	output.PrintLog(fmt.Sprintf("%s Configured git to use certificate: %s", ui.IconCheckMark, certificatePath))

	return nil
}

// CalculateGitContentType returns the type of the git test source
// Deprecated: use git instead
func (f Fetcher) CalculateGitContentType(repo testkube.Repository) (string, error) {
	if repo.Uri == "" || repo.Path == "" {
		return "", errors.New("repository uri and path should be populated")
	}

	dir, err := os.MkdirTemp("", "temp-git-files")
	if err != nil {
		return "", fmt.Errorf("could not create temporary directory for CalculateGitContentType: %s", err.Error())
	}
	// this will not overwrite the original path given how we used a value receiver for this function
	f.path = dir
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			output.PrintLog(fmt.Sprintf("%s Could not clean up after CalculateGitContentType: %s", ui.IconWarning, err.Error()))
		}
	}()

	path, err := f.FetchGitFile(&repo)
	if err != nil {
		return "", fmt.Errorf("could not fetch from git: %w", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("could not check path %s: %w", path, err)
	}

	if fileInfo.IsDir() {
		return string(testkube.TestContentTypeGitDir), nil
	}
	return string(testkube.TestContentTypeGitFile), nil
}

// GetPathAndWorkingDir returns path to git based file or dir saved in local temp directory and working dir
func GetPathAndWorkingDir(content *testkube.TestContent, dataDir string) (path, workingDir string, err error) {
	basePath, err := filepath.Abs(dataDir)
	if err != nil {
		basePath = dataDir
	}
	if content != nil {
		switch content.Type_ {
		case string(testkube.TestContentTypeString), string(testkube.TestContentTypeFileURI):
			path = filepath.Join(basePath, "test-content")
		case string(testkube.TestContentTypeGitFile), string(testkube.TestContentTypeGitDir), string(testkube.TestContentTypeGit):
			path = filepath.Join(basePath, "repo")
			if content.Repository != nil {
				if content.Repository.WorkingDir != "" {
					workingDir = filepath.Join(path, content.Repository.WorkingDir)
				}
				path = filepath.Join(path, content.Repository.Path)
			}
		}
	}

	return path, workingDir, err
}
