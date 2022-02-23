package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveAnalyticsEnabled(t *testing.T) {

	// create homedir config file
	s := Storage{}
	s.Init()

	t.Run("check if analytics system is enabled", func(t *testing.T) {
		// given / when
		d, err := Load()

		// then
		assert.NoError(t, err)
		assert.Equal(t, d.AnalyticsEnabled, true)

	})

	t.Run("check if analytics system is disabled", func(t *testing.T) {
		// given
		d, err := Load()
		assert.NoError(t, err)

		d.DisableAnalytics()
		err = Save(d)
		assert.NoError(t, err)

		// when
		d, err = Load()

		// then
		assert.NoError(t, err)
		assert.Equal(t, d.AnalyticsEnabled, false)

	})

}
