// TODO consider checkout by some go-git library to limit external deps in docker container
// in time writing code "github.com/src-d/go-git" didn't have any filter options
// "github.com/go-git/go-git/v5" also
package git

import (
	"os"

	"github.com/kubeshop/testkube/pkg/process"
)

func CheckoutCommit(uri, path, commit, dir string) error {
	_, err := process.ExecuteInDir(
		dir,
		"git",
		"init",
	)
	if err != nil {
		return err
	}

	_, err = process.ExecuteInDir(
		dir,
		"git",
		"remote",
		"add",
		"origin",
		uri,
	)
	if err != nil {
		return err
	}

	_, err = process.ExecuteInDir(
		dir,
		"git",
		"--depth",
		"1",
		"origin",
		commit,
	)
	if err != nil {
		return err
	}

	if path != "" {
		_, err = process.ExecuteInDir(
			dir,
			"git",
			"sparse-checkout",
			"set",
			path,
		)
		if err != nil {
			return err
		}
	}

	_, err = process.ExecuteInDir(
		dir,
		"git",
		"checkout",
		"FETCH_HEAD",
	)
	if err != nil {
		return err
	}

	return nil
}

// Checkout will checkout directory from Git repository
func Checkout(uri, branch, commit, dir string) (outputDir string, err error) {
	tmpDir := dir
	if tmpDir == "" {
		tmpDir, err = os.MkdirTemp("", "git-checkout")
		if err != nil {
			return "", err
		}
	}

	if commit == "" {
		args := []string{"clone"}
		if branch != "" {
			args = append(args, "-b", branch)
		}

		args = append(args, "--depth", "1", uri, "repo")
		_, err = process.ExecuteInDir(
			tmpDir,
			"git",
			args...,
		)
		if err != nil {
			return "", err
		}
	} else {

	}

	return tmpDir + "/repo/", nil
}

// PartialCheckout will checkout only given directory from Git repository
func PartialCheckout(uri, path, branch, commit, dir string) (outputDir string, err error) {
	tmpDir := dir
	if tmpDir == "" {
		tmpDir, err = os.MkdirTemp("", "git-sparse-checkout")
		if err != nil {
			return "", err
		}
	}

	if commit == "" {
		args := []string{"clone"}
		if branch != "" {
			args = append(args, "-b", branch)
		}

		args = append(args, "--depth", "1", "--filter", "blob:none", "--sparse", uri, "repo")
		_, err = process.ExecuteInDir(
			tmpDir,
			"git",
			args...,
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
	} else {

	}

	return tmpDir + "/repo/" + path, nil
}
