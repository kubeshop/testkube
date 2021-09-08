package version

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

}
