package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveClientType(t *testing.T) {
	assert.Equal(t, "proxy", (&Data{}).ResolveClientType("proxy", false))
	assert.Equal(t, "direct", (&Data{Client: "direct"}).ResolveClientType("proxy", false))
	assert.Equal(t, "cluster", (&Data{Client: "direct"}).ResolveClientType("cluster", true))
	// An explicit --client=proxy still wins over the saved client.
	assert.Equal(t, "proxy", (&Data{Client: "direct"}).ResolveClientType("proxy", true))
}

func TestValidateClientType(t *testing.T) {
	for _, c := range []string{"proxy", "direct", "cluster"} {
		assert.NoError(t, ValidateClientType(c))
	}
	assert.ErrorContains(t, ValidateClientType("cloud"), "invalid client")
	assert.Error(t, ValidateClientType(""))
}

func TestClientPersistence(t *testing.T) {
	dir := t.TempDir()
	orig := defaultDirectory
	defaultDirectory = dir
	t.Cleanup(func() {
		defaultDirectory = orig
		_ = os.RemoveAll(dir)
	})

	assert.NoError(t, Save(Data{Client: "direct"}))
	data, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "direct", data.Client)
}
