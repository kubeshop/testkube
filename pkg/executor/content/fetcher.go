package content

import (
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
	case testkube.TestContentTypeEmpty:
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
	uri, err := f.gitURI(repo)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to fetch git dir: %s", ui.IconCross, err.Error()))
		return path, err
	}

	// if path not set make full repo checkout
	if repo.Path == "" || repo.WorkingDir != "" {
		path, err := git.Checkout(uri, repo.Branch, repo.Commit, f.path)
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Failed to fetch git dir: %s", ui.IconCross, err.Error()))
			return path, fmt.Errorf("failed to fetch git dir: %w", err)
		}
		output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
		return path, nil
	}

	path, err = git.PartialCheckout(uri, repo.Path, repo.Branch, repo.Commit, f.path)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to do partial checkout on git dir: %s", ui.IconCross, err.Error()))
		return path, fmt.Errorf("failed to do partial checkout on git dir: %w", err)
	}
	output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
	return path, nil
}

// FetchGitFile returns path to git based file saved in local temp directory
func (f Fetcher) FetchGitFile(repo *testkube.Repository) (path string, err error) {
	uri, err := f.gitURI(repo)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to fetch git file: %s", ui.IconCross, err.Error()))
		return path, err
	}

	repoPath, err := git.Checkout(uri, repo.Branch, repo.Commit, f.path)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to checkout git file: %s", ui.IconCross, err.Error()))
		return path, err
	}

	path = filepath.Join(repoPath, repo.Path)
	output.PrintLog(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))
	return path, nil
}

// gitUri merge creds with git uri
func (f Fetcher) gitURI(repo *testkube.Repository) (uri string, err error) {
	if repo.Username != "" || repo.Token != "" {
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
