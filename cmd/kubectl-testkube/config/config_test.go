package config

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveAnalyticsEnabled(t *testing.T) {

	dir, err := ioutil.TempDir("", "test-config-save")
	assert.NoError(t, err)

	// create homedir config file
	s := Storage{Dir: dir}
	s.Init()

	t.Run("check if analytics system is enabled", func(t *testing.T) {
		// given / when
		d, err := s.Load()

		// then
		assert.NoError(t, err)
		assert.Equal(t, true, d.AnalyticsEnabled)

	})

	t.Run("check if analytics system is disabled", func(t *testing.T) {
		// given
		d, err := s.Load()
		assert.NoError(t, err)

		d.DisableAnalytics()
		err = s.Save(d)
		assert.NoError(t, err)

		// when
		d, err = s.Load()

		// then
		assert.NoError(t, err)
		assert.Equal(t, false, d.AnalyticsEnabled)

	})

}
