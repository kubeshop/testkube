package problem

import (
	"github.com/moogar0880/problems"
)

// Porblem is struct defining RFC7807 Problem Details
type Problem = problems.Problem

func New(status int, details string) Problem {
	pr := problems.NewDetailedProblem(status, details)
	return *pr
}
