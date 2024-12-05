package testworkflowexecutor

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/secretmanager"
)

type executionID = string
type secretName = string
type secretData = map[string]string

type SensitiveDataHandler interface {
	Process(execution *IntermediateExecution) error
	Rollback(id string) error
}

type secretHandler struct {
	manager secretmanager.SecretManager
	maps    map[executionID]map[secretName]secretData
}

func NewSecretHandler(manager secretmanager.SecretManager) *secretHandler {
	return &secretHandler{
		manager: manager,
		maps:    make(map[executionID]map[secretName]secretData),
	}
}

func (s *secretHandler) Process(intermediate *IntermediateExecution) error {
	id := intermediate.ID()

	// Pack the sensitive data into a secrets set
	secretsBatch := s.manager.Batch("twe-", id).ForceEnable()
	credentialExpressions := map[string]expressions.Expression{}
	for k, v := range intermediate.SensitiveData() {
		envVarSource, err := secretsBatch.Append(k, v)
		if err != nil {
			return err
		}
		credentialExpressions[k] = expressions.MustCompile(fmt.Sprintf(`secret("%s","%s",true)`, envVarSource.SecretKeyRef.Name, envVarSource.SecretKeyRef.Key))
	}
	secrets := secretsBatch.Get()
	s.maps[id] = make(map[string]map[string]string, len(secrets))
	for j := range secrets {
		s.maps[id][secrets[j].Name] = secrets[j].StringData
	}

	// Change the calls to access these secrets
	return intermediate.RewriteSensitiveDataCall(func(name string) (expressions.Expression, error) {
		if expr, ok := credentialExpressions[name]; ok {
			return expr, nil
		}
		return nil, fmt.Errorf(`unknown sensitive data: '%s'`, name)
	})
}

func (s *secretHandler) Get(id executionID) map[secretName]secretData {
	return s.maps[id]
}

func (s *secretHandler) Rollback(_ executionID) error {
	// There are no actual resources created, so nothing needs to be rolled back
	return nil
}
