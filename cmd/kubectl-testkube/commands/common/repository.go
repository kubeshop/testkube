package common

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func hasGitParamsInCmd(cmd *cobra.Command, crdOnly bool) bool {
	var fields = []string{"git-uri", "git-branch", "git-commit", "git-path",
		"git-username-secret", "git-token-secret", "git-working-dir", "git-certificate-secret", "git-auth-type"}
	if !crdOnly {
		fields = append(fields, "git-username", "git-token")
	}

	for _, field := range fields {
		if cmd.Flag(field).Changed {
			return true
		}
	}

	return false
}

// NewRepositoryFromFlags creates repository from command flags
func NewRepositoryFromFlags(cmd *cobra.Command) (repository *testkube.Repository, err error) {
	crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
	if err != nil {
		return nil, err
	}

	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitCommit := cmd.Flag("git-commit").Value.String()
	gitPath := cmd.Flag("git-path").Value.String()

	var gitUsername, gitToken string
	if !crdOnly {
		client, _, err := GetClient(cmd)
		ui.ExitOnError("getting client", err)

		info, err := client.GetServerInfo()
		ui.ExitOnError("getting server info", err)

		if !info.DisableSecretCreation {
			gitUsername = cmd.Flag("git-username").Value.String()
			gitToken = cmd.Flag("git-token").Value.String()
		}
	}

	gitUsernameSecret, err := cmd.Flags().GetStringToString("git-username-secret")
	if err != nil {
		return nil, err
	}

	gitTokenSecret, err := cmd.Flags().GetStringToString("git-token-secret")
	if err != nil {
		return nil, err
	}

	gitWorkingDir := cmd.Flag("git-working-dir").Value.String()
	gitCertificateSecret := cmd.Flag("git-certificate-secret").Value.String()
	gitAuthType := cmd.Flag("git-auth-type").Value.String()

	hasGitParams := hasGitParamsInCmd(cmd, crdOnly)
	if !hasGitParams {
		return nil, nil
	}

	repository = &testkube.Repository{
		Type_:             "git",
		Uri:               gitUri,
		Branch:            gitBranch,
		Commit:            gitCommit,
		Path:              gitPath,
		Username:          gitUsername,
		Token:             gitToken,
		CertificateSecret: gitCertificateSecret,
		WorkingDir:        gitWorkingDir,
		AuthType:          gitAuthType,
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

// NewRepositoryUpdateFromFlags creates repository update from command flags
func NewRepositoryUpdateFromFlags(cmd *cobra.Command) (repository *testkube.RepositoryUpdate, err error) {
	repository = &testkube.RepositoryUpdate{}

	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"git-uri",
			&repository.Uri,
		},
		{
			"git-branch",
			&repository.Branch,
		},
		{
			"git-commit",
			&repository.Commit,
		},
		{
			"git-path",
			&repository.Path,
		},
		{
			"git-working-dir",
			&repository.WorkingDir,
		},
		{
			"git-certificate-secret",
			&repository.CertificateSecret,
		},
		{
			"git-auth-type",
			&repository.AuthType,
		},
	}

	client, _, err := GetClient(cmd)
	ui.ExitOnError("getting client", err)

	info, err := client.GetServerInfo()
	ui.ExitOnError("getting server info", err)

	if !info.DisableSecretCreation {
		fields = append(fields, []struct {
			name        string
			destination **string
		}{
			{
				"git-username",
				&repository.Username,
			},
			{
				"git-token",
				&repository.Token,
			}}...)
	}

	var nonEmpty bool
	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
			nonEmpty = true
		}
	}

	var refs = []struct {
		name        string
		destination ***testkube.SecretRef
	}{
		{
			"git-username-secret",
			&repository.UsernameSecret,
		},
		{
			"git-token-secret",
			&repository.TokenSecret,
		},
	}

	for _, ref := range refs {
		if cmd.Flag(ref.name).Changed {
			value, err := cmd.Flags().GetStringToString(ref.name)
			if err != nil {
				return nil, err
			}

			for key, val := range value {
				secret := &testkube.SecretRef{
					Name: key,
					Key:  val,
				}

				*ref.destination = &secret
				nonEmpty = true
			}
		}
	}

	if nonEmpty {
		return repository, nil
	}

	return nil, nil
}

// ValidateUpsertOptions validates upsert options
func ValidateUpsertOptions(cmd *cobra.Command, sourceName string) error {
	crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
	if err != nil {
		return err
	}

	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitCommit := cmd.Flag("git-commit").Value.String()

	var gitUsername, gitToken string
	if !crdOnly {
		client, _, err := GetClient(cmd)
		ui.ExitOnError("getting client", err)

		info, err := client.GetServerInfo()
		ui.ExitOnError("getting server info", err)

		if !info.DisableSecretCreation {
			gitUsername = cmd.Flag("git-username").Value.String()
			gitToken = cmd.Flag("git-token").Value.String()
		}
	}

	gitUsernameSecret, err := cmd.Flags().GetStringToString("git-username-secret")
	if err != nil {
		return err
	}

	gitTokenSecret, err := cmd.Flags().GetStringToString("git-token-secret")
	if err != nil {
		return err
	}

	gitAuthType := cmd.Flag("git-auth-type").Value.String()
	file := cmd.Flag("file").Value.String()
	uri := cmd.Flag("uri").Value.String()

	hasGitParams := hasGitParamsInCmd(cmd, crdOnly)
	if hasGitParams && uri != "" {
		return fmt.Errorf("found git params and `--uri` flag, please use `--git-uri` for git based repo or `--uri` without git based params")
	}

	if hasGitParams && file != "" {
		return fmt.Errorf("found git params and `--file` flag, please use `--git-uri` for git based repo or `--file` without git based params")
	}

	if file != "" && uri != "" {
		return fmt.Errorf("please pass only one of `--file` and `--uri`")
	}

	if hasGitParams {
		if gitUri == "" && sourceName == "" {
			return fmt.Errorf("please pass valid `--git-uri` flag")
		}
		if gitBranch != "" && gitCommit != "" {
			return fmt.Errorf("please pass only one of `--git-branch` or `--git-commit`")
		}
	}

	if len(gitUsernameSecret) > 1 {
		return fmt.Errorf("please pass only one secret reference for git username")
	}

	if len(gitTokenSecret) > 1 {
		return fmt.Errorf("please pass only one secret reference for git token")
	}

	if gitAuthType != string(testkube.GitAuthTypeBasic) && gitAuthType != string(testkube.GitAuthTypeHeader) {
		return fmt.Errorf("please pass one of `basic` or `header` for git auth type")
	}

	if (gitUsername != "" || gitToken != "") && (len(gitUsernameSecret) > 0 || len(gitTokenSecret) > 0) {
		return fmt.Errorf("please pass only one auth method for git repository")
	}

	return nil
}
