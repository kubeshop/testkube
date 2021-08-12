package problem

import (
	"github.com/moogar0880/problems"
)

type Problem problems.DefaultProblem

func New(status int, details string) Problem {
	pr := problems.NewDetailedProblem(status, details)
	return Problem(*pr)
}
