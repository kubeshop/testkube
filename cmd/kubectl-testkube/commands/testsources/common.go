package testsources

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/spf13/cobra"

	apiclientv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func newSourceFromFlags(cmd *cobra.Command) (source *testkube.TestSource, err error) {
	var fileContent []byte

	sourceType := cmd.Flag("source-type").Value.String()
	file := cmd.Flag("file").Value.String()
	uri := cmd.Flag("uri").Value.String()
	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitCommit := cmd.Flag("git-commit").Value.String()
	gitPath := cmd.Flag("git-path").Value.String()
	gitUsername := cmd.Flag("git-username").Value.String()
	gitToken := cmd.Flag("git-token").Value.String()
	gitUsernameSecret, err := cmd.Flags().GetStringToString("git-username-secret")
	if err != nil {
		return source, err
	}

	gitTokenSecret, err := cmd.Flags().GetStringToString("git-token-secret")
	if err != nil {
		return source, err
	}

	gitWorkingDir := cmd.Flag("git-workin-dir").Value.String()
	// get file content
	if file != "" {
		fileContent, err = os.ReadFile(file)
		if err != nil {
			return source, fmt.Errorf("reading file "+file+" error: %w", err)
		}
	} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		fileContent, err = io.ReadAll(os.Stdin)
		if err != nil {
			return source, fmt.Errorf("reading stdin error: %w", err)
		}
	}

	// content is correct when is passed from file, by uri, ur by git repo
	if len(fileContent) == 0 && uri == "" && gitUri == "" {
		return source, fmt.Errorf("empty test source, please pass some data to create test source")
	}

	if gitUri != "" && sourceType == "" {
		sourceType = string(testkube.TestContentTypeGitDir)
	}

	if uri != "" && sourceType == "" {
		sourceType = string(testkube.TestContentTypeFileURI)
	}

	if len(fileContent) > 0 {
		sourceType = string(testkube.TestContentTypeString)
	}

	var repository *testkube.Repository
	if gitUri != "" {
		if sourceType == "" {
			sourceType = "git-dir"
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
	}

	source = &testkube.TestSource{
		Type_:      sourceType,
		Data:       string(fileContent),
		Repository: repository,
		Uri:        uri,
	}

	return source, nil
}

func NewUpsertTestSourceOptionsFromFlags(cmd *cobra.Command, testLabels map[string]string) (options apiclientv1.UpsertTestSourceOptions, err error) {
	source, err := newSourceFromFlags(cmd)
	ui.ExitOnError("creating source from passed parameters", err)

	name := cmd.Flag("name").Value.String()
	labels, err := cmd.Flags().GetStringToString("label")
	if err != nil {
		return options, err
	}

	options = apiclientv1.UpsertTestSourceOptions{
		Name:       name,
		Type_:      source.Type_,
		Data:       source.Data,
		Repository: source.Repository,
		Uri:        source.Uri,
	}

	// if labels are passed and are different from the existing overwrite
	if len(labels) > 0 && !reflect.DeepEqual(testLabels, labels) {
		options.Labels = labels
	} else {
		options.Labels = testLabels
	}

	return options, nil
}
