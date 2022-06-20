// TODO consider checkout by some go-git library to limit external deps in docker container
// in time writing code "github.com/src-d/go-git" didn't have any filter options
// "github.com/go-git/go-git/v5" also
package git

import (
	"os"

	"github.com/kubeshop/testkube/pkg/process"
)

// Checkout will checkout directory from Git repository
func Checkout(uri, branch, dir string) (outputDir string, err error) {
	tmpDir := dir
	if tmpDir == "" {
		tmpDir, err = os.MkdirTemp("", "git-checkout")
		if err != nil {
			return "", err
		}
	}

	_, err = process.ExecuteInDir(
		tmpDir,
		"git",
		"clone",
		"-b", branch,
		"--depth", "1",
		uri, "repo",
	)
	if err != nil {
		return "", err
	}

	return tmpDir + "/repo/", nil
}

// PartialCheckout will checkout only given directory from Git repository
func PartialCheckout(uri, path, branch, dir string) (outputDir string, err error) {
	tmpDir := dir
	if tmpDir == "" {
		tmpDir, err = os.MkdirTemp("", "git-sparse-checkout")
		if err != nil {
			return "", err
		}
	}

	_, err = process.ExecuteInDir(
		tmpDir,
		"git",
		"clone",
		"-b", branch,
		"--depth", "1",
		"--filter", "blob:none",
		"--sparse",
		uri, "repo",
	)
	if err != nil {
		return "", err
	}

	_, err = process.ExecuteInDir(
		tmpDir+"/repo",
		"git",
		"sparse-checkout",
		"set",
		path,
	)
	if err != nil {
		return "", err
	}

	return tmpDir + "/repo/" + path, nil
}
