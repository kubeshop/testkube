package orchestration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// collectPIDs returns every non-virtual pid reachable below p.
func collectPIDs(p *processNode) map[int32]bool {
	out := map[int32]bool{}
	for n := range p.nodes {
		if n.pid != -1 {
			out[n.pid] = true
		}
		for k := range collectPIDs(n) {
			out[k] = true
		}
	}
	return out
}

// TestProcessNodeChildrenOf verifies childrenOf returns only the target's
// descendant subtree, excluding the target itself and any sibling.
func TestProcessNodeChildrenOf(t *testing.T) {
	// root(-1) -> a(10) -> { a1(11) -> a11(13), a2(12) }
	//          -> b(20) -> b1(21)
	a11 := &processNode{pid: 13, nodes: map[*processNode]struct{}{}}
	a1 := &processNode{pid: 11, nodes: map[*processNode]struct{}{a11: {}}}
	a2 := &processNode{pid: 12, nodes: map[*processNode]struct{}{}}
	a := &processNode{pid: 10, nodes: map[*processNode]struct{}{a1: {}, a2: {}}}
	b1 := &processNode{pid: 21, nodes: map[*processNode]struct{}{}}
	b := &processNode{pid: 20, nodes: map[*processNode]struct{}{b1: {}}}
	root := &processNode{pid: -1, nodes: map[*processNode]struct{}{a: {}, b: {}}}

	// childrenOf(10) is exactly a's descendants: never a itself, never a sibling.
	got := collectPIDs(root.childrenOf(10))
	assert.Equal(t, map[int32]bool{11: true, 12: true, 13: true}, got)
	assert.NotContains(t, got, int32(10), "must not include the target process itself")
	assert.NotContains(t, got, int32(20), "must not include a sibling")
	assert.NotContains(t, got, int32(21), "must not include a sibling's child")

	// A leaf has no descendants.
	assert.Empty(t, collectPIDs(root.childrenOf(13)))

	// An unknown pid scopes to nothing, never falling back to the whole tree.
	assert.Empty(t, collectPIDs(root.childrenOf(999)))
}
