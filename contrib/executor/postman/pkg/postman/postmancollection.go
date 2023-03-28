package postman

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Detector is detector adapter for Postman collection saved as JSON content
type Detector struct {
}

const (
	// PostmanCollectionType is type of Postman collection
	PostmanCollectionType = "postman/collection"
)

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(options.Content.Data), &data)
	if err != nil {
		return
	}

	if info, ok := data["info"]; ok {
		if id, ok := info.(map[string]interface{})["_postman_id"]; ok && id != "" {
			return d.GetType(), true
		}
	}

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && (ext == ".json")
	return
}

func checkName(filename, pattern string) (string, bool) {
	ok := filepath.Ext(filename) == ".json" &&
		strings.HasSuffix(strings.TrimSuffix(filename, filepath.Ext(filename)), pattern)
	if !ok {
		return "", false
	}
	return filepath.Base(strings.TrimSuffix(strings.TrimSuffix(filename, filepath.Ext(filename)), pattern)), true
}

// IsTestName detecs if filename has a conventional test name
func (d Detector) IsTestName(filename string) (string, bool) {
	return checkName(filename, ".postman_collection")
}

// IsEnvName detecs if filename has a conventional env name
func (d Detector) IsEnvName(filename string) (string, string, bool) {
	filename, found := checkName(filename, ".postman_environment")
	if !found {
		return "", "", false
	}

	names := strings.Split(filename, ".")
	if len(names) != 2 {
		return "", "", false
	}

	return names[0], names[1], true
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d Detector) IsSecretEnvName(filename string) (string, string, bool) {
	filename, found := checkName(filename, ".postman_secret_environment")
	if !found {
		return "", "", false
	}

	names := strings.Split(filename, ".")
	if len(names) != 2 {
		return "", "", false
	}

	return names[0], names[1], true
}

// GetSecretVariables retuns secret variables
func (d Detector) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	var envFile EnvFile
	if err := json.Unmarshal([]byte(data), &envFile); err != nil {
		return nil, err
	}

	vars := make(map[string]testkube.Variable, 0)
	for _, value := range envFile.Values {
		references := strings.Split(value.Value, "=")
		if len(references) != 2 || !value.Enabled {
			continue
		}

		vars[value.Key] = testkube.Variable{
			Name:  value.Key,
			Type_: testkube.VariableTypeSecret,
			SecretRef: &testkube.SecretRef{
				Name: references[0],
				Key:  references[1],
			},
		}
	}

	return vars, nil
}

// GetType returns test type
func (d Detector) GetType() string {
	return PostmanCollectionType
}

// EnvFile contains env file structure
type EnvFile struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Values               []Value   `json:"values"`
	PostmanVariableScope string    `json:"_postman_variable_scope"`
	PostmanExportedAt    time.Time `json:"_postman_exported_at"`
	PostmanExportedUsing string    `json:"_postman_exported_using"`
}

// Value contains value structure
type Value struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}
