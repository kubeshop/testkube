package scripts

import (
	"io/ioutil"
	"os"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/test/script/detector"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	name         string
	file         string
	executorType string
	uri          string
	gitBranch    string
	gitPath      string
)

func NewCreateScriptsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create new script",
		Long:  `Create new Script Custom Resource, `,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			var content []byte
			var err error

			if file != "" {
				// read script content
				content, err = ioutil.ReadFile(file)
				ui.ExitOnError("reading file"+file, err)
			} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
				content, err = ioutil.ReadAll(os.Stdin)
				ui.ExitOnError("reading stdin", err)
			}

			client, namespace := GetClient(cmd)

			script, _ := client.GetScript(name)
			if name == script.Name {
				ui.Failf("Script with name '%s' already exists in namespace %s", name, namespace)
			}

			if len(content) == 0 && len(uri) == 0 {
				ui.Failf("Empty script content. Please pass some script content to create script")
			}

			var repository *testkube.Repository
			if uri != "" && gitBranch != "" {
				repository = &testkube.Repository{
					Type_:  "git",
					Uri:    uri,
					Branch: gitBranch,
					Path:   gitPath,
				}
			}

			options := apiClient.CreateScriptOptions{
				Name:       name,
				Type_:      executorType,
				Content:    string(content),
				Namespace:  namespace,
				Repository: repository,
			}

			// try to detect type if none passed
			if executorType == "" {
				d := detector.NewDefaultDetector()
				if detectedType, ok := d.Detect(options); ok {
					ui.Info("Detected test script type", detectedType)
					options.Type_ = detectedType
				}
			}

			if options.Type_ == "" {
				ui.Failf("Can't detect executor type by passed file content, please pass valid --type flag")
			}

			script, err = client.CreateScript(options)
			ui.ExitOnError("creating script "+name+" in namespace "+namespace, err)

			ui.Success("Script created", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique script name - mandatory")
	cmd.Flags().StringVarP(&file, "file", "f", "", "script file - will be read from stdin if not specified")

	cmd.Flags().StringVarP(&executorType, "type", "t", "", "script type (defaults to postman-collection)")

	cmd.Flags().StringVarP(&uri, "uri", "", "", "if resource need to be loaded from URI")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")

	return cmd
}
