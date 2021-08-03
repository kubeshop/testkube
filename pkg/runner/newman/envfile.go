package newman

import (
	"bytes"
	"encoding/json"
	"io"
	"time"
)

func NewEnvFileReader(m map[string]string) (io.Reader, error) {
	envFile := NewEnvFile(m)
	b, err := json.Marshal(envFile)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), err
}

func NewEnvFile(m map[string]string) (envFile EnvFile) {
	envFile.ID = "executor-env-file"
	envFile.Name = "executor-env-file"
	envFile.PostmanVariableScope = "environment"
	envFile.PostmanExportedAt = time.Now()
	envFile.PostmanExportedUsing = "Postman/7.34.0"

	for k, v := range m {
		envFile.Values = append(envFile.Values, Value{Key: k, Value: v, Enabled: true})
	}

	return
}

type EnvFile struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Values               []Value   `json:"values"`
	PostmanVariableScope string    `json:"_postman_variable_scope"`
	PostmanExportedAt    time.Time `json:"_postman_exported_at"`
	PostmanExportedUsing string    `json:"_postman_exported_using"`
}

type Value struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}
