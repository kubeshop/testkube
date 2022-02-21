package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveAnalyticsEnabled(t *testing.T) {
	// init
	c := config{}

	// create homedir
	c.Init()

	c.EnableAnalytics()
	c.Save(c.Data)

	t.Run("check if analytics system is enabled", func(t *testing.T) {
		// given
		c := config{}

		// when
		d, err := c.Load()

		// then
		assert.NoError(t, err)
		assert.Equal(t, d.AnalyticsEnabled, true)

	})

	t.Run("check if analytics system is enabled", func(t *testing.T) {
		// given
		c := config{}
		c.DisableAnalytics()
		err := c.Save(c.Data)

		assert.NoError(t, err)

		// when
		d, err := c.Load()

		// then
		assert.NoError(t, err)
		assert.Equal(t, d.AnalyticsEnabled, false)

	})

}
