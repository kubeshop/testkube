package postman

import "encoding/json"

type ExecuteRequest struct {
	Type     string          `json:"type,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}
