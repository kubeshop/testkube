package localinstall

import (
	"errors"
	"strings"

	licensevalidator "github.com/kubeshop/testkube/pkg/diagnostics/validators/license"
)

const (
	MaxLicenseAttempts = 3

	LicenseHelp            = "Find your key in the email from Testkube, or get one at https://testkube.io/get-started/on-prem"
	LicenseUnreachableHelp = "Testkube checks licenses at license.testkube.io and needs it to run.\nCheck your network, proxy or firewall, then run again"
)

var (
	ErrLicenseMissing     = errors.New("no license key entered")
	ErrLicenseInvalid     = errors.New("license key is not valid")
	ErrLicenseUnreachable = errors.New("cannot reach license.testkube.io")
)

type KeyPrompter interface {
	Ask(attempt int) (string, error)
}

type licenseValidator interface {
	Validate(key string) (valid bool, err error)
}

// LicenseAttempt is one try, kept so each can be tracked.
type LicenseAttempt struct {
	Number int
	Status string
}

type LicenseStep struct {
	validator licenseValidator
	prompter  KeyPrompter
}

func NewLicenseStep(prompter KeyPrompter) *LicenseStep {
	return &LicenseStep{validator: licenseService{client: licensevalidator.NewClient()}, prompter: prompter}
}

// A flag key can't be retyped: one try only.
func (s *LicenseStep) Run(flagKey string) (key string, attempts []LicenseAttempt, err error) {
	maxAttempts := MaxLicenseAttempts
	if flagKey != "" {
		maxAttempts = 1
	}
	for n := 1; n <= maxAttempts; n++ {
		key = strings.TrimSpace(flagKey)
		if flagKey == "" {
			answer, err := s.prompter.Ask(n)
			if err != nil {
				return "", attempts, err
			}
			key = strings.TrimSpace(answer)
		}
		if key == "" {
			return "", append(attempts, LicenseAttempt{Number: n, Status: "missing"}), ErrLicenseMissing
		}
		valid, err := s.validator.Validate(key)
		switch {
		case err != nil:
			// Retrying won't help; the cluster would fail the same way.
			return "", append(attempts, LicenseAttempt{Number: n, Status: "unreachable"}), ErrLicenseUnreachable
		case valid:
			return key, append(attempts, LicenseAttempt{Number: n, Status: "valid"}), nil
		}
		attempts = append(attempts, LicenseAttempt{Number: n, Status: "invalid"})
	}
	return "", attempts, ErrLicenseInvalid
}

type licenseService struct {
	client *licensevalidator.Client
}

func (s licenseService) Validate(key string) (bool, error) {
	resp, err := s.client.ValidateLicense(licensevalidator.LicenseRequest{License: key})
	if err != nil {
		return false, err
	}
	return resp.Valid, nil
}
