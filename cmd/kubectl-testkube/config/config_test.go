package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSave(t *testing.T) {
	// override default directory
	dir, err := os.MkdirTemp("", "test-config-save")
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	defaultDirectory = dir

	t.Run("test save into default storage", func(t *testing.T) {
		err := Save(Data{})
		assert.NoError(t, err)
	})

	t.Run("test load stored config into default storage", func(t *testing.T) {
		err := Save(Data{Namespace: "some", TelemetryEnabled: true})
		assert.NoError(t, err)
		data, err := Load()
		assert.NoError(t, err)
		assert.Equal(t, "some", data.Namespace)
		assert.Equal(t, true, data.TelemetryEnabled)
	})
}

func TestDatabaseTypeConstants(t *testing.T) {
	assert.Equal(t, "mongodb", DatabaseTypeMongoDB)
	assert.Equal(t, "postgresql", DatabaseTypePostgreSQL)
	assert.NotEqual(t, DatabaseTypeMongoDB, DatabaseTypePostgreSQL)
}

func TestCloudContextDatabaseType_Persistence(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-config-dbtype")
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	defaultDirectory = dir

	t.Run("persist MongoDB database type", func(t *testing.T) {
		data := Data{
			CloudContext: CloudContext{
				DatabaseType: DatabaseTypeMongoDB,
			},
		}
		err := Save(data)
		assert.NoError(t, err)

		loaded, err := Load()
		assert.NoError(t, err)
		assert.Equal(t, DatabaseTypeMongoDB, loaded.CloudContext.DatabaseType)
	})

	t.Run("persist PostgreSQL database type", func(t *testing.T) {
		data := Data{
			CloudContext: CloudContext{
				DatabaseType: DatabaseTypePostgreSQL,
			},
		}
		err := Save(data)
		assert.NoError(t, err)

		loaded, err := Load()
		assert.NoError(t, err)
		assert.Equal(t, DatabaseTypePostgreSQL, loaded.CloudContext.DatabaseType)
	})

	t.Run("empty database type on fresh config", func(t *testing.T) {
		data := Data{}
		err := Save(data)
		assert.NoError(t, err)

		loaded, err := Load()
		assert.NoError(t, err)
		assert.Empty(t, loaded.CloudContext.DatabaseType)
	})
}

func TestCloudContextAgentRelease_Persistence(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-config-agentrelease")
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	defaultDirectory = dir

	t.Run("persist agent release name and namespace", func(t *testing.T) {
		data := Data{
			CloudContext: CloudContext{
				AgentReleaseName: "testkube-my-agent",
				AgentNamespace:   "testkube-agents",
			},
		}
		err := Save(data)
		assert.NoError(t, err)

		loaded, err := Load()
		assert.NoError(t, err)
		assert.Equal(t, "testkube-my-agent", loaded.CloudContext.AgentReleaseName)
		assert.Equal(t, "testkube-agents", loaded.CloudContext.AgentNamespace)
	})

	t.Run("empty agent release on fresh config", func(t *testing.T) {
		data := Data{}
		err := Save(data)
		assert.NoError(t, err)

		loaded, err := Load()
		assert.NoError(t, err)
		assert.Empty(t, loaded.CloudContext.AgentReleaseName)
		assert.Empty(t, loaded.CloudContext.AgentNamespace)
	})
}

func TestSaveTelemetryEnabled(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-config-save")
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	// create homedir config file
	s := Storage{Dir: dir}
	err = s.Init()
	assert.NoError(t, err)

	t.Run("check if telemetry system is enabled", func(t *testing.T) {
		// given / when
		d, err := s.Load()

		// then
		assert.NoError(t, err)
		assert.Equal(t, true, d.TelemetryEnabled)
	})

	t.Run("check if telemetry system is disabled", func(t *testing.T) {
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
		assert.Equal(t, false, d.TelemetryEnabled)
	})
}
