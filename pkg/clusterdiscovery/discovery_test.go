package clusterdiscovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	authzv1 "k8s.io/api/authorization/v1"
)

func TestCanWatch(t *testing.T) {
	rules := []authzv1.ResourceRule{
		{Verbs: []string{"list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods"}},
		{Verbs: []string{"*"}, APIGroups: []string{"cert-manager.io"}, Resources: []string{"certificates"}},
		{Verbs: []string{"list", "watch"}, APIGroups: []string{"*"}, Resources: []string{"*"}},
	}

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, canWatch(rules[:1], "", "pods"))
	})
	t.Run("wildcard verb", func(t *testing.T) {
		assert.True(t, canWatch(rules[1:2], "cert-manager.io", "certificates"))
	})
	t.Run("wildcard group and resource", func(t *testing.T) {
		assert.True(t, canWatch(rules[2:3], "kafka.strimzi.io", "kafkatopics"))
	})
	t.Run("no matching rule", func(t *testing.T) {
		assert.False(t, canWatch(rules[:2], "batch", "jobs"))
	})
	t.Run("list-only rule without watch", func(t *testing.T) {
		listOnly := []authzv1.ResourceRule{
			{Verbs: []string{"list"}, APIGroups: []string{""}, Resources: []string{"pods"}},
		}
		assert.False(t, canWatch(listOnly, "", "pods"))
	})
}

func TestContainsOrWildcard(t *testing.T) {
	assert.True(t, containsOrWildcard([]string{"get", "list", "watch"}, "watch"))
	assert.True(t, containsOrWildcard([]string{"*"}, "watch"))
	assert.False(t, containsOrWildcard([]string{"get", "list"}, "watch"))
	assert.False(t, containsOrWildcard(nil, "watch"))
}
