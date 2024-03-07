// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCloneCmd() *cobra.Command {
	var (
		paths    []string
		username string
		token    string
		authType string
		revision string
	)

	cmd := &cobra.Command{
		Use:   "clone <uri> <outputPath>",
		Short: "Clone the Git repository",
		Args:  cobra.ExactArgs(2),

		Run: func(cmd *cobra.Command, args []string) {
			uri, err := url.Parse(args[0])
			ui.ExitOnError("repository uri", err)
			outputPath, err := filepath.Abs(args[1])
			ui.ExitOnError("output path", err)

			// Disable interactivity
			os.Setenv("GIT_TERMINAL_PROMPT", "0")

			authArgs := make([]string, 0)

			if authType == "header" {
				ui.Debug("auth type: header")
				if token != "" {
					authArgs = append(authArgs, "-c", fmt.Sprintf("http.extraHeader='%s'", "Authorization: Bearer "+token))
				}
				if username != "" {
					uri.User = url.User(username)
				}
			} else {
				ui.Debug("auth type: token")
				if username != "" && token != "" {
					uri.User = url.UserPassword(username, token)
				} else if username != "" {
					uri.User = url.User(username)
				} else if token != "" {
					uri.User = url.User(token)
				}
			}

			// Mark directory as safe
			configArgs := []string{"-c", fmt.Sprintf("safe.directory=%s", outputPath), "-c", "advice.detachedHead=false"}

			// Clone repository
			if len(paths) == 0 {
				ui.Debug("full checkout")
				err = Run("git", "clone", configArgs, authArgs, "--depth", 1, "--verbose", uri.String(), outputPath)
				ui.ExitOnError("cloning repository", err)
			} else {
				ui.Debug("sparse checkout")
				err = Run("git", "clone", configArgs, authArgs, "--filter=blob:none", "--no-checkout", "--sparse", "--depth", 1, "--verbose", uri.String(), outputPath)
				ui.ExitOnError("cloning repository", err)
				err = Run("git", "-C", outputPath, configArgs, "sparse-checkout", "set", "--no-cone", paths)
				ui.ExitOnError("sparse checkout repository", err)
				if revision != "" {
					err = Run("git", "-C", outputPath, configArgs, "fetch", authArgs, "--depth", 1, "origin", revision)
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
		},
	}

	cmd.Flags().StringSliceVarP(&paths, "paths", "p", nil, "paths for sparse checkout")
	cmd.Flags().StringVarP(&username, "username", "u", "", "")
	cmd.Flags().StringVarP(&token, "token", "t", "", "")
	cmd.Flags().StringVarP(&authType, "authType", "a", "basic", "allowed: basic, header")
	cmd.Flags().StringVarP(&revision, "revision", "r", "", "commit hash, branch name or tag")

	return cmd
}
