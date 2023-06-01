package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListenerName(t *testing.T) {
	t.Parallel()
	// given
	in := "webhooks.http://localhost:8080/something/else.start-test"

	// when
	result := ListenerName(in)

	// then
	assert.Equal(t, "webhooks.httplocalhost8080somethingelse.starttest", result)

}
