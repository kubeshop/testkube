package ai

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/sashabaranov/go-openai"
)

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		ApiKey: apiKey,
	}
}

type OpenAI struct {
	ApiKey string
}

func (a OpenAI) AnalyzeTestExecution(ctx context.Context, execution testkube.Execution) (string, error) {

	message := `Act as software engineer who need to debug failing test. Test is written in "` + execution.TestType + `" and it fails.`
	message += "\n\n"
	message += "Test status is: " + string(*execution.ExecutionResult.Status) + ".\n"
	message += `Test runs against Service run Kubernetes, test itself is run in Testkube.\n`
	message += `Test execution results are like follows:\n`

	message += execution.ExecutionResult.Output

	message += "\n\n"
	message += `What should I do to make the test pass?`

	ai := openai.NewClient(a.ApiKey)
	resp, err := ai.CreateChatCompletion(
		ctx,
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

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
