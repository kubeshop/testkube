package license

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexp(t *testing.T) {
	key := "8D9E74-444441-DEF387-31EE50-5F2A2E-V3"
	match, _ := regexp.MatchString(`^([^-]{6}-){5}[^-]{2}$`, key)
	assert.True(t, match)
}
