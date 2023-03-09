package testsources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiclientv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func newSourceFromFlags(cmd *cobra.Command) (source *testkube.TestSource, err error) {
	sourceType := cmd.Flag("source-type").Value.String()
	uri := cmd.Flag("uri").Value.String()
	gitUri := cmd.Flag("git-uri").Value.String()

	data, err := common.NewDataFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	fileContent := ""
	if data != nil {
		fileContent = *data
	}

	// content is correct when is passed from file, by uri, ur by git repo
	if len(fileContent) == 0 && uri == "" && gitUri == "" {
		return nil, fmt.Errorf("empty test source, please pass some data to create test source")
	}

	if uri != "" && sourceType == "" {
		sourceType = string(testkube.TestContentTypeFileURI)
	}

	if len(fileContent) > 0 && sourceType == "" {
		sourceType = string(testkube.TestContentTypeString)
	}

	repository, err := common.NewRepositoryFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	if repository != nil && sourceType == "" {
		sourceType = string(testkube.TestContentTypeGit)
	}

	source = &testkube.TestSource{
		Type_:      sourceType,
		Data:       string(fileContent),
		Repository: repository,
		Uri:        uri,
	}

	return source, nil
}

// NewUpsertTestSourceOptionsFromFlags creates test source options from command flags
func NewUpsertTestSourceOptionsFromFlags(cmd *cobra.Command) (options apiclientv1.UpsertTestSourceOptions, err error) {
	source, err := newSourceFromFlags(cmd)
	if err != nil {
		return options, fmt.Errorf("creating source from passed parameters %w", err)
	}

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
		Labels:     labels,
	}

	return options, nil
}

func newSourceUpdateFromFlags(cmd *cobra.Command) (source *testkube.TestSourceUpdate, err error) {
	source = &testkube.TestSourceUpdate{}

	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"source-type",
			&source.Name,
		},
		{
			"uri",
			&source.Uri,
		},
	}

	var nonEmpty bool
	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
			nonEmpty = true
		}
	}

	data, err := common.NewDataFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	if data != nil {
		source.Data = data
		nonEmpty = true
	}

	repository, err := common.NewRepositoryUpdateFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	if repository != nil {
		source.Repository = &repository
		nonEmpty = true
	}

	if nonEmpty {
		return source, nil
	}

	return nil, nil
}

// NewUpdateTestSourceOptionsFromFlags creates test update source options from command flags
func NewUpdateTestSourceOptionsFromFlags(cmd *cobra.Command) (options apiclientv1.UpdateTestSourceOptions, err error) {
	source, err := newSourceUpdateFromFlags(cmd)
	if err != nil {
		return options, fmt.Errorf("creating source from passed parameters %w", err)
	}

	if cmd.Flag("name").Changed {
		name := cmd.Flag("name").Value.String()
		options.Name = &name
	}

	if cmd.Flag("label").Changed {
		labels, err := cmd.Flags().GetStringToString("label")
		if err != nil {
			return options, err
		}

		options.Labels = &labels
	}

	if source != nil {
		options.Type_ = source.Type_
		var emptyValue string
		var emptyRepository = &testkube.RepositoryUpdate{}
		switch {
		case source.Data != nil:
			options.Data = source.Data
			options.Repository = &emptyRepository
			options.Uri = &emptyValue
		case source.Repository != nil:
			options.Data = &emptyValue
			options.Repository = source.Repository
			options.Uri = &emptyValue
		case source.Uri != nil:
			options.Data = &emptyValue
			options.Repository = &emptyRepository
			options.Uri = source.Uri
		}
	}

	return options, nil
}
