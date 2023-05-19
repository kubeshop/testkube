package ai

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ai"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"

	openai "github.com/sashabaranov/go-openai"
)

var (
	executionID string
)

func NewResultsAnalysisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze <executionName>",
		Short: "List artifacts of the given test or test suite execution name",
		Args:  validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			executionID = args[0]
			cmd.SilenceUsage = true
			client, _ := common.GetClient(cmd)
			execution, err := client.GetExecution(executionID)
			ui.ExitOnError("getting execution", err)

			if *execution.ExecutionResult.Status != *testkube.ExecutionStatusFailed {
				ui.Failf("Test is not failing, so there is no need to debug it with AI")
				return
			}

			ui.H1("Debugging test results with AI " + openai.GPT3Dot5Turbo)

			ui.Properties([][]string{
				{"Test name", execution.TestName},
				{"Test type", execution.TestType},
				{"Test status", ui.Red(*execution.ExecutionResult.Status)},
			})

			ui.H2("AI Analysis:")
			stream, err := ai.NewOpenAI(os.Getenv("OPENAI_KEY")).AnalyzeTestExecutionStream(ctx, execution)
			ui.ExitOnError("analyzing test execution", err)

			for resp := range stream {
				ui.Print(resp)
			}

			ui.NL(2)
			ui.Success("Done")
		},
	}

	return cmd
}
