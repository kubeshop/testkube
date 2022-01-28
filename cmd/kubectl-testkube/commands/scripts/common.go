package scripts

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"time"

	apiclientv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/runner/output"
	"github.com/kubeshop/testkube/pkg/test/script/detector"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func printExecutionDetails(execution testkube.Execution) {
	ui.Warn("Type          :", execution.ScriptType)
	ui.Warn("Name          :", execution.ScriptName)
	ui.Warn("Execution ID  :", execution.Id)
	ui.Warn("Execution name:", execution.Name)
	ui.NL()
	ui.NL()
}

func DownloadArtifacts(id, dir string, client apiclientv1.Client) {
	artifacts, err := client.GetExecutionArtifacts(id)
	ui.ExitOnError("getting artifacts ", err)

	err = os.MkdirAll(dir, os.ModePerm)
	ui.ExitOnError("creating dir "+dir, err)

	if len(artifacts) > 0 {
		ui.Info("Getting artifacts", fmt.Sprintf("count = %d", len(artifacts)), "\n")
	}
	for _, artifact := range artifacts {
		f, err := client.DownloadFile(id, artifact.Name, dir)
		ui.ExitOnError("downloading file: "+f, err)
		ui.Warn(" - downloading file ", f)
	}

	ui.NL()
	ui.NL()
}

func watchLogs(id string, client apiclientv1.Client) {
	ui.Info("Getting pod logs")

	logs, err := client.Logs(id)
	ui.ExitOnError("getting logs from executor", err)

	for l := range logs {
		switch l.Type_ {
		case output.TypeError:
			ui.Errf(l.Content)
			if l.Result != nil {
				ui.Errf("Error: %s", l.Result.ErrorMessage)
				ui.Debug("Output: %s", l.Result.Output)
			}
			uiShellGetExecution(id)
			os.Exit(1)
			return
		case output.TypeResult:
			ui.Info("Execution completed", l.Result.Output)
		default:
			ui.LogLine(l.String())
		}
	}

	ui.NL()

	// TODO watch for success | error status - in case of connection error on logs watch need fix in 0.8
	for range time.Tick(time.Second) {
		execution, err := client.GetExecution(id)
		ui.ExitOnError("get script execution details", err)

		fmt.Print(".")

		if execution.ExecutionResult.IsCompleted() {
			fmt.Println()

			uiShellGetExecution(id)

			return
		}
	}

	uiShellGetExecution(id)
}

func newContentFromFlags(cmd *cobra.Command) (content *testkube.ScriptContent, err error) {
	var fileContent []byte

	file := cmd.Flag("file").Value.String()
	uri := cmd.Flag("uri").Value.String()
	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitPath := cmd.Flag("git-path").Value.String()
	gitUsername := cmd.Flag("git-username").Value.String()
	gitToken := cmd.Flag("git-token").Value.String()

	if file != "" {
		fileContent, err = ioutil.ReadFile(file)
		return content, fmt.Errorf("reading file "+file+" error: %w", err)
	} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		fileContent, err = ioutil.ReadAll(os.Stdin)
		return content, fmt.Errorf("reading stdin error: %w", err)
	}

	if len(fileContent) == 0 && len(uri) == 0 {
		return content, fmt.Errorf("empty script content, please pass some script content to create script")
	}

	var repository *testkube.Repository
	if uri != "" && gitBranch != "" {
		repository = &testkube.Repository{
			Type_:    "git",
			Uri:      gitUri,
			Branch:   gitBranch,
			Path:     gitPath,
			Username: gitUsername,
			Token:    gitToken,
		}
	}

	content = &testkube.ScriptContent{
		Data:       string(fileContent),
		Repository: repository,
		Uri:        uri,
	}

	return content, nil
}

func NewUpsertScriptOptionsFromFlags(cmd *cobra.Command, script testkube.Script) (options apiclientv1.UpsertScriptOptions, err error) {
	content, err := newContentFromFlags(cmd)
	ui.ExitOnError("creating content from passed parameters", err)

	name := cmd.Flag("name").Value.String()
	executorType := cmd.Flag("type").Value.String()
	namespace := cmd.Flag("script-namespace").Value.String()
	tags, err := cmd.Flags().GetStringSlice("tags")
	if err != nil {
		return options, err
	}

	options = apiclientv1.UpsertScriptOptions{
		Name:      name,
		Type_:     executorType,
		Content:   content,
		Namespace: namespace,
	}

	// if tags are passed and are different from the existing overwrite
	if len(tags) > 0 && !reflect.DeepEqual(script.Tags, tags) {
		options.Tags = tags
	} else {
		options.Tags = script.Tags
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
		return options, fmt.Errorf("Can't detect executor type by passed file content (%s), please pass valid --type flag", executorType)
	}

	return options, nil

}
