// TODO consider checkout by some go-git library to limit external deps in docker container
// in time writing code "github.com/src-d/go-git" didn't have any filter options
// "github.com/go-git/go-git/v5" also
package git

import (
	"io/ioutil"

	"github.com/kubeshop/testkube/pkg/process"
)

// Partial checkout will checkout only given directory from Git repository
func PartialCheckout(uri, path, branch string) (outputDir string, err error) {

	tmpDir, err := ioutil.TempDir("", "testkube-scripts")
	if err != nil {
		return tmpDir, err
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
