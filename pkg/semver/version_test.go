package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersioning(t *testing.T) {

	t.Run("test get newest version with v prefix", func(t *testing.T) {
		versions := []string{
			"v0.1.1",
			"0.3.2",
			"v0.3.3",
			"0.2.2",
		}

		newest := GetNewest(versions)
		assert.Equal(t, "0.3.3", newest)

	})

	t.Run("next - bump patch", func(t *testing.T) {
		next, err := Next("0.0.5", Patch)
		assert.NoError(t, err)
		assert.Equal(t, "0.0.6", next)
	})

	t.Run("next - bump minor", func(t *testing.T) {
		next, err := Next("0.0.5", Minor)
		assert.NoError(t, err)
		assert.Equal(t, "0.1.0", next)
	})

	t.Run("next - bump major", func(t *testing.T) {
		next, err := Next("0.0.5", Major)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0", next)
	})

	t.Run("next - bump prerelease number", func(t *testing.T) {
		next, err := NextPrerelease("0.0.5-beta001")
		assert.NoError(t, err)
		assert.Equal(t, "0.0.5-beta002", next)
	})
}

func TestLte(t *testing.T) {
	t.Run("equal versions in different string forms", func(t *testing.T) {
		// Regression: "1.0" and "1.0.0" are semantically equal and must be <=.
		ok, err := Lte("1.0", "1.0.0")
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Lte("v1.2.3", "1.2.3")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("strictly less-than", func(t *testing.T) {
		ok, err := Lte("1.0.0", "1.0.1")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("greater-than is not lte", func(t *testing.T) {
		ok, err := Lte("2.0.0", "1.9.9")
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid version returns error", func(t *testing.T) {
		_, err := Lte("not-a-version", "1.0.0")
		assert.Error(t, err)
	})
}

func TestGetNewestEmpty(t *testing.T) {
	t.Run("empty list returns empty string without panicking", func(t *testing.T) {
		assert.Equal(t, "", GetNewest(nil))
		assert.Equal(t, "", GetNewest([]string{}))
	})

	t.Run("no valid versions returns empty string", func(t *testing.T) {
		assert.Equal(t, "", GetNewest([]string{"not-a-version", "also-bad"}))
	})
}

func TestNextPrerelease(t *testing.T) {

	t.Run("beta postfix", func(t *testing.T) {
		bumped, err := NextPrerelease("0.0.1-beta008")
		assert.NoError(t, err)
		assert.Equal(t, "0.0.1-beta009", bumped)
	})

	t.Run("alpha postfix", func(t *testing.T) {
		bumped, err := NextPrerelease("0.0.1-alpha009")
		assert.NoError(t, err)
		assert.Equal(t, "0.0.1-alpha010", bumped)
	})
}
