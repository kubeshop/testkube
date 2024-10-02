package problem

import (
	"github.com/moogar0880/problems"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/utils/codec"
)

// Porblem is struct defining RFC7807 Problem Details
type Problem problems.DefaultProblem

func New(status int, details string) Problem {
	pr := problems.NewDetailedProblem(status, details)
	return Problem(*pr)
}

func CommandErrorJSONBytes(command cloud.Command, status int, title string, err error) ([]byte, error) {
	var errString string
	if err != nil {
		errString = err.Error()
	}
	pr := problems.NewDetailedProblem(status, errString)
	pr.Type = "Command Error"
	pr.Title = title
	pr.Instance = string(command)

	return codec.ToJSONBytes(Problem(*pr))
}
