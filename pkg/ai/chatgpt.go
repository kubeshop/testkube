package ai

import (
	"context"
	"fmt"

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

func (a OpenAI) GenerateTest(ctx context.Context, testType string, testParameters map[string]string) (string, error) {
	var supportedTestTypes = map[string]struct {
		testEngine   string
		testArtifact string
	}{
		"k6/script": {
			testEngine:   "K6",
			testArtifact: "K6 script file",
		},
		"postman/collection": {
			testEngine:   "Newman",
			testArtifact: "Postam json collection",
		},
	}

	testDetails, ok := supportedTestTypes[testType]
	if !ok {
		return "", fmt.Errorf("not supported test type %s", testType)
	}

	message := fmt.Sprintf("Act as software engineer who needs to develop a smoke test and run it using %s tool.\n", testDetails.testEngine)
	message += "Test is run against Service in Kubernetes, test itself is run by Testkube.\n"
	message += fmt.Sprintf("Write the actual %s for this test.\n", testDetails.testArtifact)
	message += "Test parameters are as follows: \n"
	for key, value := range testParameters {
		message += fmt.Sprintf("Parameter name: %s and parameter value: %s \n", key, value)
	}

	ai := openai.NewClient(a.ApiKey)
	resp, err := ai.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			MaxTokens:   2048,
			Temperature: 0.7,
			Model:       openai.GPT3Dot5Turbo,
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

	fmt.Println("\n\n", resp.Choices)

	return resp.Choices[0].Message.Content, nil
}
