package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	assertion := require.New(t)

	cfg, err := Get()
	if err != nil {
		t.Errorf("Get() failed, expected nil, got %v", err)
	}

	assertion.NoError(err)
	assertion.IsType(&Config{}, cfg)
}
