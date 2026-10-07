package localinstall

import (
	"errors"
	"strings"
	"time"

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
	Validate(key string) (valid bool, expiry time.Time, err error)
}

// Expiry is zero when the service sends none.
type License struct {
	Key    string
	Expiry time.Time
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
func (s *LicenseStep) Run(flagKey string) (license License, attempts []LicenseAttempt, err error) {
	maxAttempts := MaxLicenseAttempts
	if flagKey != "" {
		maxAttempts = 1
	}
	for n := 1; n <= maxAttempts; n++ {
		key := strings.TrimSpace(flagKey)
		if flagKey == "" {
			answer, err := s.prompter.Ask(n)
			if err != nil {
				return License{}, attempts, err
			}
			key = strings.TrimSpace(answer)
		}
		if key == "" {
			return License{}, append(attempts, LicenseAttempt{Number: n, Status: "missing"}), ErrLicenseMissing
		}
		valid, expiry, err := s.validator.Validate(key)
		switch {
		case err != nil:
			// Retrying won't help; the cluster would fail the same way.
			return License{}, append(attempts, LicenseAttempt{Number: n, Status: "unreachable"}), ErrLicenseUnreachable
		case valid:
			return License{Key: key, Expiry: expiry}, append(attempts, LicenseAttempt{Number: n, Status: "valid"}), nil
		}
		attempts = append(attempts, LicenseAttempt{Number: n, Status: "invalid"})
	}
	return License{}, attempts, ErrLicenseInvalid
}

type licenseService struct {
	client *licensevalidator.Client
}

func (s licenseService) Validate(key string) (bool, time.Time, error) {
	resp, err := s.client.ValidateLicense(licensevalidator.LicenseRequest{License: key})
	if err != nil {
		return false, time.Time{}, err
	}
	// Display only: an odd format must not fail validation.
	expiry, _ := time.Parse(time.RFC3339, resp.License.Expiry)
	return resp.Valid, expiry, nil
}
