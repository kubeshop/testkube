package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRoundDuration(t *testing.T) {
	t.Run("round duration should round to default 100ms", func(t *testing.T) {

		//given
		d := time.Duration(99111111111)

		// when
		rounded := RoundDuration(d)

		// then
		assert.Equal(t, "1m39.11s", rounded.String())
	})

	t.Run("round duration should round to given value", func(t *testing.T) {

		//given
		d := time.Duration(99111111111)

		// when
		rounded := RoundDuration(d, time.Minute)

		// then
		assert.Equal(t, "2m0s", rounded.String())
	})
}
