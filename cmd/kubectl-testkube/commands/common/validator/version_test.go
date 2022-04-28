package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimPatchVersion(t *testing.T) {
	t.Run("trims release version", func(t *testing.T) {
		v := TrimPatchVersion("1.0.3")
		assert.Equal(t, "1.0.0", v)
	})

	t.Run("trims dev version", func(t *testing.T) {
		v := TrimPatchVersion("1.0.3-jd3jd939jd9j")
		assert.Equal(t, "1.0.0", v)
	})
}

func TestValidateVersions(t *testing.T) {

	t.Run("versions with different patch number should be ok", func(t *testing.T) {
		apiVersion := "1.10.3"
		clientVersion := "1.10.22"

		err := ValidateVersions(apiVersion, clientVersion)

		assert.NoError(t, err)
	})

	t.Run("versions with different minor number should not be ok", func(t *testing.T) {
		apiVersion := "1.11.1"
		clientVersion := "1.10.1"

		err := ValidateVersions(apiVersion, clientVersion)

		assert.ErrorIs(t, err, ErrOldClientVersion)
	})

}
