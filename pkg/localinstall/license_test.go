package localinstall

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakePrompter struct {
	answers []string
	asked   int
}

func (f *fakePrompter) Ask(int) (string, error) {
	answer := f.answers[f.asked]
	f.asked++
	return answer, nil
}

type fakeValidator struct {
	valid map[string]bool
	err   error
	calls int
}

func (f *fakeValidator) Validate(key string) (bool, time.Time, error) {
	f.calls++
	return f.valid[key], time.Time{}, f.err
}

func TestLicenseStep_Run(t *testing.T) {
	tests := []struct {
		name          string
		flagKey       string
		answers       []string
		validatorErr  error
		wantKey       string
		wantErr       error
		wantStatuses  []string
		wantAsked     int
		wantValidated int
	}{
		{"empty input stops at once without validating", "", []string{"  "}, nil, "", ErrLicenseMissing, []string{"missing"}, 1, 0},
		{"three wrong keys stop", "", []string{"a", "b", "c"}, nil, "", ErrLicenseInvalid, []string{"invalid", "invalid", "invalid"}, 3, 3},
		{"right key on second try", "", []string{"bad", " good "}, nil, "good", nil, []string{"invalid", "valid"}, 2, 2},
		{"unreachable service does not retry", "", []string{"good"}, errors.New("dial tcp: timeout"), "", ErrLicenseUnreachable, []string{"unreachable"}, 1, 1},
		{"wrong flag key is not retried by prompting", "bad", nil, nil, "", ErrLicenseInvalid, []string{"invalid"}, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompter := &fakePrompter{answers: tt.answers}
			validator := &fakeValidator{valid: map[string]bool{"good": true}, err: tt.validatorErr}
			step := &LicenseStep{validator: validator, prompter: prompter}

			license, attempts, err := step.Run(tt.flagKey)

			assert.Equal(t, tt.wantKey, license.Key)
			assert.ErrorIs(t, err, tt.wantErr)
			var statuses []string
			for _, a := range attempts {
				statuses = append(statuses, a.Status)
			}
			assert.Equal(t, tt.wantStatuses, statuses)
			assert.Equal(t, tt.wantAsked, prompter.asked)
			assert.Equal(t, tt.wantValidated, validator.calls)
		})
	}
}
