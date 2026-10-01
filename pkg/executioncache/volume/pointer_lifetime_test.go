package volume

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The bucket-wide rule is left unfiltered on purpose, so it expires cache objects too.
// Reading only the cache rule would let the sweep delete entries weeks before the
// pointers naming them went away, and every hit on those keys would then restore
// nothing - for as long as the pointer survived, with no run able to replace it.
func TestPointerLifetimeCountsTheBucketWideRule(t *testing.T) {
	ttl, expires := PointerLifetime(0, 30)

	assert.True(t, expires)
	assert.Equal(t, 30*24*time.Hour, ttl)
}

// Where both rules match one object the store applies the earlier, so that is the one
// the retention has to clear.
func TestPointerLifetimeTakesTheEarlierRule(t *testing.T) {
	ttl, expires := PointerLifetime(3, 30)
	assert.True(t, expires)
	assert.Equal(t, 3*24*time.Hour, ttl)

	ttl, expires = PointerLifetime(30, 3)
	assert.True(t, expires)
	assert.Equal(t, 3*24*time.Hour, ttl)
}

// Nothing expiring pointers is not a duration, and must not be mistaken for one.
func TestPointerLifetimeReportsWhenNothingExpiresAPointer(t *testing.T) {
	ttl, expires := PointerLifetime(0, 0)

	assert.False(t, expires)
	assert.Zero(t, ttl)
}
