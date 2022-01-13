package testkube

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewStartedTestExecution(name string) TestExecution {
	return TestExecution{
		Id:        primitive.NewObjectID().Hex(),
		StartTime: time.Now(),
		Name:      name,
		Status:    TestStatusQueued,
	}
}

func (e TestExecution) Table() (header []string, output [][]string) {
	header = []string{"Step", "Status", "Error", "ID"}
	output = make([][]string, 0)

	// TODO introduce Array ArrayHeader? interface to allow easily compose array like data in model
	for _, sr := range e.StepResults {
		switch sr.Step.Type() {
		case TestStepTypeExecuteScript:
			status := "unknown"
			id := ""
			errorMessage := ""
			if sr.Execution != nil && sr.Execution.ExecutionResult != nil && sr.Execution.ExecutionResult.Status != nil {
				status = string(*sr.Execution.ExecutionResult.Status)
				errorMessage = sr.Execution.ExecutionResult.ErrorMessage
				id = sr.Execution.Id
			} else {
				status = "no execution results"
			}

			row := []string{sr.Step.FullName(), status, errorMessage, id}
			output = append(output, row)
		case TestStepTypeDelay:
			row := []string{sr.Step.FullName(), "success", "", ""}
			output = append(output, row)
		}
	}

	return
}
