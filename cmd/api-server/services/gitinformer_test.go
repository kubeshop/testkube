package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	intconfig "github.com/kubeshop/testkube/internal/config"
)

func TestShouldRunGitInformer(t *testing.T) {
	t.Run("enabled in OSS mode when control plane triggers are off", func(t *testing.T) {
		assert.True(t, ShouldRunGitInformer(false, false, intconfig.ProContext{}))
	})

	t.Run("enabled in OSS mode without environment id", func(t *testing.T) {
		assert.True(t, ShouldRunGitInformer(true, false, intconfig.ProContext{}))
	})

	t.Run("disabled in cloud mode without environment id", func(t *testing.T) {
		assert.False(t, ShouldRunGitInformer(true, true, intconfig.ProContext{}))
	})

	t.Run("enabled in cloud mode with environment id", func(t *testing.T) {
		assert.True(t, ShouldRunGitInformer(true, true, intconfig.ProContext{
			EnvID: "tkcenv_123",
		}))
	})
}
