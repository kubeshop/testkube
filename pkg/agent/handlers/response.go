package handlers

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/problem"
	"github.com/kubeshop/testkube/pkg/utils/codec"
)

func BadRequestResponse(c cloud.Command, messageId string, err error) *cloud.ExecuteResponse {
	return &cloud.ExecuteResponse{MessageId: messageId, Status: 400, Body: errToJSONProblemBytes(c, 400, "Bad Request", err)}
}

func NotFoundResponse(c cloud.Command, messageId string, err error) *cloud.ExecuteResponse {
	return &cloud.ExecuteResponse{MessageId: messageId, Status: 404, Body: errToJSONProblemBytes(c, 404, "Not Found", err)}
}

func InternalServerErrorResponse(c cloud.Command, messageId string, err error) *cloud.ExecuteResponse {
	return &cloud.ExecuteResponse{MessageId: messageId, Status: 500, Body: errToJSONProblemBytes(c, 500, "", err)}
}

func SuccessResponse(c cloud.Command, messageId string, body any) *cloud.ExecuteResponse {
	bytes, err := codec.ToJSONBytes(body)
	if err != nil {
		return &cloud.ExecuteResponse{MessageId: messageId, Status: 500, Body: errToJSONProblemBytes(c, 500, "can't encode response data to JSON", err)}
	}

	return &cloud.ExecuteResponse{MessageId: messageId, Status: 200, Body: bytes}
}

func errToJSONProblemBytes(c cloud.Command, status int, title string, err error) []byte {
	var pr problem.Problem
	body, err := problem.CommandErrorJSONBytes(c, status, title, err)
	if err != nil {
		pr = problem.New(status, fmt.Sprintf("%s: %s", title, err))
	} else {
		pr = problem.Problem{Status: status, Title: title, Detail: string(body)}
	}

	bytes, err := codec.ToJSONBytes(pr)
	if err != nil {
		return []byte(fmt.Sprintf("error encoding problem to JSON: %s", err))
	}
	return bytes

}
