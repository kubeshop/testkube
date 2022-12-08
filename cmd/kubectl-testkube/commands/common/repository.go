package common

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/spf13/cobra"
)

func NewRepositoryFromFlags(cmd *cobra.Command) (repository *testkube.Repository, err error) {
	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitCommit := cmd.Flag("git-commit").Value.String()
	gitPath := cmd.Flag("git-path").Value.String()
	gitUsername := cmd.Flag("git-username").Value.String()
	gitToken := cmd.Flag("git-token").Value.String()
	gitUsernameSecret, err := cmd.Flags().GetStringToString("git-username-secret")
	if err != nil {
		return nil, err
	}

	gitTokenSecret, err := cmd.Flags().GetStringToString("git-token-secret")
	if err != nil {
		return nil, err
	}

	gitWorkingDir := cmd.Flag("git-working-dir").Value.String()

	hasGitParams := gitBranch != "" || gitCommit != "" || gitPath != "" || gitUri != "" || gitToken != "" || gitUsername != "" ||
		len(gitUsernameSecret) > 0 || len(gitTokenSecret) > 0 || gitWorkingDir != ""

	if !hasGitParams {
		return nil, nil
	}

	repository = &testkube.Repository{
		Type_:      "git",
		Uri:        gitUri,
		Branch:     gitBranch,
		Commit:     gitCommit,
		Path:       gitPath,
		Username:   gitUsername,
		Token:      gitToken,
		WorkingDir: gitWorkingDir,
	}

	for key, val := range gitUsernameSecret {
		repository.UsernameSecret = &testkube.SecretRef{
			Name: key,
			Key:  val,
		}
	}

	for key, val := range gitTokenSecret {
		repository.TokenSecret = &testkube.SecretRef{
			Name: key,
			Key:  val,
		}
	}

	return repository, nil
}

func NewRepositoryUpdateFromFlags(cmd *cobra.Command) (repository *testkube.RepositoryUpdate, err error) {
	repository = &testkube.RepositoryUpdate{}

	var fields = []struct {
		name        string
		destination *string
	}{
		{
			"git-uri",
			repository.Uri,
		},
		{
			"git-branch",
			repository.Branch,
		},
		{
			"git-commit",
			repository.Commit,
		},
		{
			"git-path",
			repository.Path,
		},
		{
			"git-username",
			repository.Username,
		},
		{
			"git-token",
			repository.Token,
		},
		{
			"git-working-dir",
			repository.WorkingDir,
		},
	}

	var nonEmpty bool
	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			field.destination = &value
			nonEmpty = true
		}
	}

	if cmd.Flag("git-username-secret").Changed {
		gitUsernameSecret, err := cmd.Flags().GetStringToString("git-username-secret")
		if err != nil {
			return nil, err
		}

		for key, val := range gitUsernameSecret {
			secret := &testkube.SecretRef{
				Name: key,
				Key:  val,
			}

			repository.UsernameSecret = &secret
			nonEmpty = true
		}
	}

	if cmd.Flag("git-token-secret").Changed {
		gitTokenSecret, err := cmd.Flags().GetStringToString("git-token-secret")
		if err != nil {
			return nil, err
		}

		for key, val := range gitTokenSecret {
			secret := &testkube.SecretRef{
				Name: key,
				Key:  val,
			}

			repository.TokenSecret = &secret
			nonEmpty = true
		}
	}

	if !nonEmpty {
		return repository, nil
	}

	return nil, nil
}
