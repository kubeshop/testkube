package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	t.Parallel()

	os.Setenv("NATS_URI", "aaa")
	defer os.Unsetenv("NATS_URI")
	cfg, err := Get()
	assert.Equal(t, "aaa", cfg.NatsURI)
	assert.NoError(t, err, "getting config not return error")
}
