package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

const (
	LicenseEndpoint = "https://license.testkube.io/owner" //this is default but it can be set using ldflag -X github.com/kubeshop/testkube/pkg/telemetry.LicenseEndpoint=https://license.localhost
)

type EmailResponse struct {
	Owner struct {
		Email string `json:"email"`
	} `json:"owner"`
}
type EmailRequest struct {
	License string `json:"license"`
}

// GetEmail returns email
func GetEmail(license string) string {
	if LicenseEndpoint != "" {
		payload := EmailRequest{License: license}
		jsonPayload, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, LicenseEndpoint, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return ""
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return ""
		}
		defer resp.Body.Close()

		var emailResponse EmailResponse
		err = json.NewDecoder(resp.Body).Decode(&emailResponse)
		if err != nil {
			return ""
		}
		return emailResponse.Owner.Email
	}
	return ""
}
