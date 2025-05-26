package commands

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	CloneRetryOnFailureMaxAttempts = 5
	CloneRetryOnFailureBaseDelay   = 100 * time.Millisecond
)

var (
	protocolRe = regexp.MustCompile(`^[^:]+://`)
)

func NewCloneCmd() *cobra.Command {
	var (
		rawPaths []string
		username string
		token    string
		sshKey   string
		authType string
		revision string
		cone     bool
	)

	cmd := &cobra.Command{
		Use:   "clone <uri> <outputPath>",
		Short: "Clone the Git repository",
		Args:  cobra.ExactArgs(2),

		Run: func(cmd *cobra.Command, args []string) {
			// Append SSH protocol if there is missing one and it looks like that (git@github.com:kubeshop/testkube.git)
			if !protocolRe.MatchString(args[0]) && strings.ContainsRune(args[0], ':') && !strings.ContainsRune(args[0], '\\') {
				args[0] = "ssh://" + strings.Replace(args[0], ":", "/", 1)
			}
			uri, err := url.Parse(args[0])
			ui.ExitOnError("repository uri", err)
			destinationPath, err := filepath.Abs(args[1])
			ui.ExitOnError("output path", err)

			// Disable interactivity
			os.Setenv("GIT_TERMINAL_PROMPT", "0")

			// Clean paths for sparse checkout to make them more compliant with Git requirements
			paths := make([]string, 0)
			for _, p := range rawPaths {
				p = filepath.Clean(p)
				if cone && p != "/" && strings.HasPrefix(p, "/") {
					// Delete leading '/' for cone
					p = p[1:]
				}
				if p != "" && p != "." {
					paths = append(paths, p)
				}
			}

			authArgs := make([]string, 0)

			if authType == "header" {
				ui.Debug("auth type: header")
				if token != "" {
					authArgs = append(authArgs, "-c", fmt.Sprintf("http.extraHeader='%s'", "Authorization: Bearer "+token))
				}
				if username != "" {
					uri.User = url.User(username)
				}
			} else if authType == "github" {
				client, err := env.Cloud()
				if err != nil {
					ui.Failf("could not create cloud client: %v", err)
				}
				githubToken, err := client.GetGitHubToken(cmd.Context(), uri.String())
				if err == nil {
					uri.User = url.UserPassword("x-access-token", githubToken)
				}
			} else {
				ui.Debug("auth type: basic")
				if username != "" && token != "" {
					uri.User = url.UserPassword(username, token)
				} else if username != "" {
					uri.User = url.User(username)
				} else if token != "" {
					uri.User = url.User(token)
				}
			}

			// Use the SSH key (ensure there is new line at EOF)
			sshKey = strings.TrimRight(sshKey, "\n") + "\n"
			if sshKey != "\n" {
				sshKeyPath := filepath.Join(constants.DefaultTmpDirPath, "id_rsa")
				err := os.WriteFile(sshKeyPath, []byte(sshKey), 0400)
				ui.ExitOnError("saving SSH key temporarily", err)
				os.Setenv("GIT_SSH_COMMAND", shellquote.Join("ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-i", sshKeyPath))
			}

			// Keep the files in temporary directory
			outputPath := filepath.Join(constants.DefaultTmpDirPath, "repo")
			// Mark directory as safe
			configArgs := []string{"-c", fmt.Sprintf("safe.directory=%s", outputPath), "-c", "advice.detachedHead=false"}

			fmt.Printf("📦 ")

			// Clone repository
			if len(paths) == 0 {
				ui.Debug("full checkout")
				if revision == "" {
					err = RunWithRetry(CloneRetryOnFailureMaxAttempts, CloneRetryOnFailureBaseDelay, "git", "clone", configArgs, authArgs, "--depth", 1, "--verbose", uri.String(), outputPath)
				} else {
					err = RunWithRetry(CloneRetryOnFailureMaxAttempts, CloneRetryOnFailureBaseDelay, "git", "clone", configArgs, authArgs, "--depth", 1, "--branch", revision, "--verbose", uri.String(), outputPath)
				}
				ui.ExitOnError("cloning repository", err)
			} else {
				ui.Debug("sparse checkout")
				err = RunWithRetry(CloneRetryOnFailureMaxAttempts, CloneRetryOnFailureBaseDelay, "git", "clone", configArgs, authArgs, "--filter=blob:none", "--no-checkout", "--sparse", "--depth", 1, "--verbose", uri.String(), outputPath)
				ui.ExitOnError("cloning repository", err)
				coneArgs := []string{"--no-cone"}
				if cone {
					coneArgs = nil
				}
				err = RunWithRetry(CloneRetryOnFailureMaxAttempts, CloneRetryOnFailureBaseDelay, "git", "-C", outputPath, configArgs, "sparse-checkout", "set", coneArgs, paths)
				ui.ExitOnError("sparse checkout repository", err)
				if revision != "" {
					err = RunWithRetry(CloneRetryOnFailureMaxAttempts, CloneRetryOnFailureBaseDelay, "git", "-C", outputPath, configArgs, "fetch", authArgs, "--depth", 1, "origin", revision)
					ui.ExitOnError("fetching revision", err)
					err = Run("git", "-C", outputPath, configArgs, "checkout", "FETCH_HEAD")
					ui.ExitOnError("checking out head", err)
					// TODO: Don't do it for commits
					err = Run("git", "-C", outputPath, configArgs, "checkout", "-B", revision)
					ui.ExitOnError("checking out the branch", err)
				} else {
					err = Run("git", "-C", outputPath, configArgs, "checkout")
					ui.ExitOnError("fetching head", err)
				}
			}

			// Copy files to the expected directory. Ignore errors, only inform warn about them.
			fmt.Printf("📥 Moving the contents to %s...\n", destinationPath)
			err = copy.Copy(outputPath, destinationPath, copy.Options{
				OnError: func(src, dest string, err error) error {
					if err != nil {
						if src == outputPath && strings.Contains(err.Error(), "chmod") {
							// Ignore chmod error on mounted directory
							return nil
						}
						fmt.Printf("warn: copying to %s: %s\n", dest, err.Error())
					}
					return nil
				},
			})
			ui.ExitOnError("copying files to destination", err)

			// Allow the default group to write in all the cloned directories
			fmt.Printf("📥 Adjusting access permissions...\n")
			err = filepath.WalkDir(destinationPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				mode := info.Mode()
				if mode.Perm()&0o060 != 0o060 {
					err := os.Chmod(path, mode|0o060) // read/write for the FS group
					if err != nil {
						fmt.Printf("warn: chmod %s: %s\n", path, err.Error())
					}
				}
				return nil
			})
			ui.ExitOnError("setting permissions", err)

			fmt.Printf("🔎 Destination folder contains following files ...\n")
			filepath.Walk(destinationPath, func(name string, info fs.FileInfo, err error) error {
				// bold the folder name
				if info.IsDir() {
					fmt.Printf("\x1b[1m%s\x1b[0m\n", name)
				} else {
					fmt.Println(name)
				}
				return nil
			})

			err = os.RemoveAll(outputPath)
			ui.ExitOnError("deleting the temporary directory", err)
		},
	}

	cmd.Flags().StringSliceVarP(&rawPaths, "paths", "p", nil, "paths for sparse checkout")
	cmd.Flags().StringVarP(&username, "username", "u", "", "")
	cmd.Flags().StringVarP(&token, "token", "t", "", "")
	cmd.Flags().StringVarP(&sshKey, "sshKey", "s", "", "")
	cmd.Flags().StringVarP(&authType, "authType", "a", "basic", "allowed: basic, header")
	cmd.Flags().StringVarP(&revision, "revision", "r", "", "commit hash, branch name or tag")
	cmd.Flags().BoolVar(&cone, "cone", false, "should enable cone mode for sparse checkout")

	return cmd
}
