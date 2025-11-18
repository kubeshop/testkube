package testkube

import (
	"github.com/kubeshop/testkube/pkg/utils"
)

type WatchTestSuiteExecutionResponse struct {
	Execution TestSuiteExecution
	Error     error
}

func (e *TestSuiteExecution) convertDots(fn func(string) string) *TestSuiteExecution {
	labels := make(map[string]string, len(e.Labels))
	for key, value := range e.Labels {
		labels[fn(key)] = value
	}
	e.Labels = labels

	envs := make(map[string]string, len(e.Envs))
	for key, value := range e.Envs {
		envs[fn(key)] = value
	}
	e.Envs = envs

	vars := make(map[string]Variable, len(e.Variables))
	for key, value := range e.Variables {
		vars[fn(key)] = value
	}
	e.Variables = vars
	return e
}

func (e *TestSuiteExecution) EscapeDots() *TestSuiteExecution {
	return e.convertDots(utils.EscapeDots)
}

func (e *TestSuiteExecution) UnscapeDots() *TestSuiteExecution {
	return e.convertDots(utils.UnescapeDots)
}

func (e *TestSuiteExecution) CleanStepsOutput() *TestSuiteExecution {
	for i := range e.StepResults {
		if e.StepResults[i].Execution != nil && e.StepResults[i].Execution.ExecutionResult != nil {
			e.StepResults[i].Execution.ExecutionResult.Output = ""
		}
	}

	for i := range e.ExecuteStepResults {
		for j := range e.ExecuteStepResults[i].Execute {
			if e.ExecuteStepResults[i].Execute[j].Execution != nil && e.ExecuteStepResults[i].Execute[j].Execution.ExecutionResult != nil {
				e.ExecuteStepResults[i].Execute[j].Execution.ExecutionResult.Output = ""
			}
		}
	}
	return e
}

func (e *TestSuiteExecution) TruncateErrorMessages(length int) *TestSuiteExecution {
	for _, bs := range e.ExecuteStepResults {
		for _, sr := range bs.Execute {
			if sr.Execution != nil && sr.Execution.ExecutionResult != nil && len(sr.Execution.ExecutionResult.ErrorMessage) > length {
				sr.Execution.ExecutionResult.ErrorMessage = sr.Execution.ExecutionResult.ErrorMessage[0:length]
			}
		}
	}
	return e
}
