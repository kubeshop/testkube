package ai

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"

	"context"

	openai "github.com/sashabaranov/go-openai"
)

var (
	executionID string
)

func NewResultsAnalysisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "results <executionName>",
		Short: "List artifacts of the given test or test suite execution name",
		Args:  validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			executionID = args[0]
			cmd.SilenceUsage = true
			client, _ := common.GetClient(cmd)
			execution, err := client.GetExecution(executionID)
			ui.ExitOnError("getting execution", err)

			test, err := client.GetTest(execution.TestName)
			ui.ExitOnError("getting test", err)
			testType := test.Type_

			if *execution.ExecutionResult.Status != *testkube.ExecutionStatusFailed {
				ui.Failf("Test is not failing, so there is no need to debug it with AI")
				return
			}

			ui.H1("Debugging test results with AI " + openai.GPT3Dot5Turbo)
			ui.Properties([][]string{
				{"Test name", execution.TestName},
				{"Test type", testType},
				{"Test status", ui.Red(*execution.ExecutionResult.Status)},
			})

			message := `Act as software engineer who need to debug failing test. Test is written in "` + testType + `" and it fails.`
			message += "\n\n"
			message += "Test status is: " + string(*execution.ExecutionResult.Status) + ".\n"
			message += `Test runs against Service run Kubernetes, test itself is run in Testkube.\n`
			message += `Test execution results are like follows:\n`

			message += execution.ExecutionResult.Output

			message += "\n\n"
			message += `What should I do to make the test pass?`

			ui.Debug("openai message ", message)

			s := ui.NewSpinner("Waiting for AI response...")

			ai := openai.NewClient(os.Getenv("OPENAI_KEY"))
			resp, err := ai.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model: openai.GPT3Dot5Turbo,
					Messages: []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleUser,
							Content: message,
						},
					},
				},
			)

			s.Success()

			ui.ExitOnError("chat completion error", err)

			ui.Info("\n\nAI response:")
			ui.Paragraph(resp.Choices[0].Message.Content)
			ui.NL(2)

		},
	}

	return cmd
}
