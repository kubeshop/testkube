package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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

func (a OpenAI) AnalyzeTestExecutionStream(ctx context.Context, execution testkube.Execution) (chan string, error) {

	out := make(chan string)

	message := `Act as software engineer who need to debug failing test. Test is written in "` + execution.TestType + `" and it fails.`
	message += "\n\n"
	message += "Test status is: " + string(*execution.ExecutionResult.Status) + ".\n"
	message += `Test runs against Service run Kubernetes, test itself is run in Testkube.\n`
	message += `Test execution results are like follows:\n`

	message += execution.ExecutionResult.Output

	message += "\n\n"
	message += `What should I do to make the test pass?`

	ai := openai.NewClient(a.ApiKey)
	stream, err := ai.CreateChatCompletionStream(
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
		return out, err
	}

	go func(out chan string) {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(out)
				return
			} else if err != nil {
				fmt.Printf("%+v\n", err)
				close(out)
			}
			out <- resp.Choices[0].Delta.Content
		}
	}(out)

	return out, nil
}
