package orchestration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessNodeSelectRoots_SelectsOnlyTrackedSubtrees(t *testing.T) {
	step2 := &processNode{pid: 2, nodes: map[*processNode]struct{}{}}
	step1 := &processNode{pid: 1, nodes: map[*processNode]struct{}{step2: {}}}
	unrelated := &processNode{pid: 3, nodes: map[*processNode]struct{}{}}
	leaf := &processNode{pid: 5, nodes: map[*processNode]struct{}{}}
	branch := &processNode{pid: 4, nodes: map[*processNode]struct{}{leaf: {}}}
	root := &processNode{pid: -1, nodes: map[*processNode]struct{}{
		step1:     {},
		unrelated: {},
		branch:    {},
	}}

	selected := root.SelectRoots([]int32{1, 5})
	require.NotNil(t, selected)
	assert.Len(t, selected.nodes, 2)
	assert.Contains(t, selected.nodes, step1)
	assert.Contains(t, selected.nodes, leaf)
	assert.NotContains(t, selected.nodes, unrelated)
}

func TestProcessNodeSelectRoots_PrefersAncestorOverTrackedDescendant(t *testing.T) {
	child := &processNode{pid: 2, nodes: map[*processNode]struct{}{}}
	parent := &processNode{pid: 1, nodes: map[*processNode]struct{}{child: {}}}
	root := &processNode{pid: -1, nodes: map[*processNode]struct{}{parent: {}}}

	selected := root.SelectRoots([]int32{2, 1})
	require.NotNil(t, selected)
	assert.Len(t, selected.nodes, 1)
	assert.Contains(t, selected.nodes, parent)
	assert.NotContains(t, selected.nodes, child)
}
