package license

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

// Test_Licenses basic test when licenses files are provided
func Test_Licenses(t *testing.T) {

	var workingLicense = os.Getenv("TEST_WORKING_LICENSE")
	var exampleWorkingFile = os.Getenv("TEST_VALID_LICENSE_FILE")
	var fileTTLExpired = os.Getenv("TEST_INVALID_LICENSE_FILE_TTL")
	var licenseExpired = os.Getenv("TEST_INVALID_LICENSE_FILE_EXPIRED")

	if workingLicense == "" {
		t.Skip("working license env var not provided skipping test")
	}

	t.Run("valid key valid file", func(t *testing.T) {
		if exampleWorkingFile == "" {
			t.Skip("env vars not provided skipping test")
		}
		v := NewOfflineLicenseValidator(workingLicense, exampleWorkingFile)
		r := v.Validate("")
		assert.Equal(t, r.Status, validators.StatusValid)
	})

	t.Run("valid key file ttl expired", func(t *testing.T) {
		if fileTTLExpired == "" {
			t.Skip("env vars not provided skipping test")
		}
		v := NewOfflineLicenseValidator(workingLicense, fileTTLExpired)
		r := v.Validate("")

		assert.Equal(t, r.Status, validators.StatusInvalid)
		assert.Equal(t, r.Errors[0].Error(), ErrOfflineLicenseFileExpired.Error())
	})

	t.Run("valid key license expired file", func(t *testing.T) {
		if licenseExpired == "" {
			t.Skip("env vars not provided skipping test")
		}
		v := NewOfflineLicenseValidator(workingLicense, licenseExpired)
		r := v.Validate("")

		assert.Equal(t, r.Status, validators.StatusInvalid)
		assert.Equal(t, r.Errors[0].Error(), ErrOfflineLicenseExpired.Error())
	})
}
