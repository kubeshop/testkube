package runner

// CurlRunnerInput is the input for the CurlRunner
type CurlRunnerInput struct {
	Command        []string `json:"command"`
	ExpectedStatus string   `json:"expected_status"`
	ExpectedBody   string   `json:"expected_body"`
}

// FillTemplates resolves the templates from the CurlRunnerInput against the values in the param
func (runnerInput *CurlRunnerInput) FillTemplates(params map[string]string) error {

	err := ResolveTemplates(runnerInput.Command, params)
	if err != nil {
		return err
	}

	runnerInput.ExpectedBody, err = ResolveTemplate(runnerInput.ExpectedBody, params)
	if err != nil {
		return err
	}

	runnerInput.ExpectedStatus, err = ResolveTemplate(runnerInput.ExpectedStatus, params)
	if err != nil {
		return err
	}

	return nil
}
