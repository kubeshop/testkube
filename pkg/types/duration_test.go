package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormattedDuration(t *testing.T) {

	t.Run("formats to default format valid duration", func(t *testing.T) {
		// given
		fd := FormattedDuration(time.Hour + time.Minute)
		// when
		formatted := fd.Format()
		// then
		assert.Equal(t, "01:01:00", formatted)
	})

	t.Run("formats to passed format valid duration", func(t *testing.T) {
		// given
		fd := FormattedDuration(time.Hour + time.Minute)
		// when
		formatted := fd.Format("3:04pm")
		// then
		assert.Equal(t, "1:01am", formatted)
	})
}

func TestFormatDuration(t *testing.T) {
	t.Run("formats to default format on valid duration", func(t *testing.T) {
		// given / when
		out := FormatDuration("1h1m1s")
		// then
		assert.Equal(t, "01:01:01", out)
	})

	t.Run("returns invalid when can't parse duration", func(t *testing.T) {
		// given / when
		out := FormatDuration("13123jj1j1j1j1j1")
		// then
		assert.Contains(t, out, InvalidDurationString)
	})
}
