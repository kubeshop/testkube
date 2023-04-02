package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckOut(t *testing.T) {
	t.Skip("TODO implement me")

	repo := "https://github.com/cirosantilli/test-git-partial-clone-big-small"

	dir, err := PartialCheckout(repo, "", "small", "master", "", "")
	t.Logf("partial repo checkedout to dir: %s", dir)
	assert.NoError(t, err)
	t.Fail()
}
