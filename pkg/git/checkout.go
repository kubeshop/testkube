// TODO consider checkout by some go-git library to limit external deps in docker container
// in time writing code "github.com/src-d/go-git" didn't have any filter options
// "github.com/go-git/go-git/v5" also
package git

import (
	"os"

	"github.com/kubeshop/testkube/pkg/process"
)

// CheckoutCommit checks out specific commit
func CheckoutCommit(uri, path, commit, dir string) (err error) {
	repoDir := dir + "/repo"
	if err = os.Mkdir(repoDir, 0750); err != nil {
		return err
	}

	if _, err = process.ExecuteInDir(
		repoDir,
		"git",
		"init",
	); err != nil {
		return err
	}

	if _, err = process.ExecuteInDir(
		repoDir,
		"git",
		"remote",
		"add",
		"origin",
		uri,
	); err != nil {
		return err
	}

	if _, err = process.ExecuteInDir(
		repoDir,
		"git",
		"fetch",
		"--depth",
		"1",
		"origin",
		commit,
	); err != nil {
		return err
	}

	if path != "" {
		if _, err = process.ExecuteInDir(
			repoDir,
			"git",
			"sparse-checkout",
			"set",
			path,
		); err != nil {
			return err
		}
	}

	if _, err = process.ExecuteInDir(
		repoDir,
		"git",
		"checkout",
		"FETCH_HEAD",
	); err != nil {
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
		if err = CheckoutCommit(uri, "", commit, tmpDir); err != nil {
			return "", err
		}
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
		if err = CheckoutCommit(uri, path, commit, tmpDir); err != nil {
			return "", err
		}
	}

	return tmpDir + "/repo/" + path, nil
}
