package ai

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ai"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"

	openai "github.com/sashabaranov/go-openai"
)

func NewTestGenerationCmd() *cobra.Command {
	var (
		testName       string
		testType       string
		testParameters map[string]string
	)

	cmd := &cobra.Command{
		Use:   "generate <testType>",
		Short: "Generate test CRD using selected test type and test parameters",
		Run: func(cmd *cobra.Command, args []string) {
			if testName == "" {
				ui.Failf("pass valid test name (in '--name' flag)")
			}

			ui.H1("Generating test with AI " + openai.GPT3Dot5Turbo)

			s := ui.NewSpinner("Generating test with AI")
			ctx := context.Background()
			resp, err := ai.NewOpenAI(os.Getenv("OPENAI_KEY")).GenerateTest(ctx, testType, testParameters)
			if err != nil {
				s.Fail(err.Error())
			} else {
				s.Success()
			}

			ui.H2("AI generation completed")
			namespace := cmd.Flag("namespace").Value.String()
			options := testkube.Test{
				Name:      testName,
				Namespace: namespace,
				Type_:     testType,
				Content:   testkube.NewStringTestContent(resp),
			}
			(*testkube.TestUpsertRequest)(&options).QuoteTestTextFields()
			data, err := crd.ExecuteTemplate(crd.TemplateTest, options)
			ui.ExitOnError("executing crd template", err)

			ui.Info(data)
		},
	}

	cmd.Flags().StringVarP(&testName, "name", "", "", "unique test name - mandatory")
	cmd.Flags().StringVarP(&testType, "type", "t", "postman/collection", "test type")
	cmd.Flags().StringToStringVarP(&testParameters, "test-parameter", "", nil, "test parameter name and value")

	return cmd
}
