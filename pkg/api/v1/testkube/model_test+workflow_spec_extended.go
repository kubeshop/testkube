package testkube

import (
	"encoding/json"
	"hash/fnv"
)

func (t TestWorkflowSpec) GetConfigHash() (uint64, error) {
	data, err := json.Marshal(t.Config)
	if err != nil {
		return 0, err
	}

	configHash := fnv.New64a()
	configHash.Write(data)

	return configHash.Sum64(), nil
}
