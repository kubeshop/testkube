// TODO consider checkout by some go-git library to limit external deps in docker container
// in time writing code "github.com/src-d/go-git" didn't have any filter options
// "github.com/go-git/go-git/v5" also
package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/process"
)

// CheckoutCommit checks out specific commit
func CheckoutCommit(uri, authHeader, path, commit, dir string) (err error) {
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

	args := []string{}
	// Appends the HTTP Authorization header to the git clone args to
	// authenticate using a bearer token. More info:
	// https://confluence.atlassian.com/bitbucketserver/http-access-tokens-939515499.html
	if authHeader != "" {
		args = append(args, "-c", fmt.Sprintf("http.extraHeader='%s'", authHeader))
	}

	args = append(args, "fetch", "--depth", "1", "origin", commit)
	_, err = process.ExecuteInDir(
		repoDir,
		"git",
		args...,
	)
	output.PrintLogf("Git parameters: %s", strings.Join(obfuscateArgs(args, uri, authHeader), " "))
	if err != nil {
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
func Checkout(uri, authHeader, branch, commit, dir string) (outputDir string, err error) {
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

		// Appends the HTTP Authorization header to the git clone args to
		// authenticate using a bearer token. More info:
		// https://confluence.atlassian.com/bitbucketserver/http-access-tokens-939515499.html
		if authHeader != "" {
			args = append(args, "-c", fmt.Sprintf("http.extraHeader='%s'", authHeader))
		}

		args = append(args, "--depth", "1", uri, "repo")
		_, err = process.ExecuteInDir(
			tmpDir,
			"git",
			args...,
		)
		output.PrintLogf("Git parameters: %s", strings.Join(obfuscateArgs(args, uri, authHeader), " "))
		if err != nil {
			return "", err
		}
	} else {
		if err = CheckoutCommit(uri, authHeader, "", commit, tmpDir); err != nil {
			return "", err
		}
	}

	return tmpDir + "/repo/", nil
}

// PartialCheckout will checkout only given directory from Git repository
func PartialCheckout(uri, authHeader, path, branch, commit, dir string) (outputDir string, err error) {
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

		// Appends the HTTP Authorization header to the git clone args to
		// authenticate using a bearer token. More info:
		// https://confluence.atlassian.com/bitbucketserver/http-access-tokens-939515499.html
		if authHeader != "" {
			args = append(args, "-c", fmt.Sprintf("http.extraHeader='%s'", authHeader))
		}

		args = append(args, "--depth", "1", "--filter", "blob:none", "--sparse", uri, "repo")
		_, err = process.ExecuteInDir(
			tmpDir,
			"git",
			args...,
		)
		output.PrintLogf("Git parameters: %s", strings.Join(obfuscateArgs(args, uri, authHeader), " "))
		if err != nil {
			return "", err
		}

		_, err = process.ExecuteInDir(
			tmpDir+"/repo",
			"git",
			"sparse-checkout",
			"set",
			"--no-cone",
			path,
		)
		if err != nil {
			return "", err
		}
	} else {
		if err = CheckoutCommit(uri, authHeader, path, commit, tmpDir); err != nil {
			return "", err
		}
	}

	return tmpDir + "/repo/" + path, nil
}

func obfuscateArgs(args []string, uri, authHeader string) []string {
	for i := range args {
		for _, value := range []string{uri, authHeader} {
			if value != "" {
				args[i] = strings.ReplaceAll(args[i], value, strings.Repeat("*", len(value)))
			}
		}
	}

	return args
}
