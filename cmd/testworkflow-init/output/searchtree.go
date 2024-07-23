package output

type SearchTree struct {
	tree map[byte]*SearchTree
	last bool
}

func NewSearchTree() *SearchTree {
	return &SearchTree{tree: map[byte]*SearchTree{}}
}

func (r *SearchTree) HasChildren() bool {
	return len(r.tree) != 0
}

func (r *SearchTree) Append(word []byte) {
	node := r
	for i := 0; i < len(word); i++ {
		if _, ok := node.tree[word[i]]; !ok {
			node.tree[word[i]] = NewSearchTree()
		}
		node = node.tree[word[i]]
	}
	node.last = true
}

func (r *SearchTree) Hits(b []byte, index int) (int, int, bool, *SearchTree) {
	// It may continue, unless some byte is not found
	end := -1
	mayContinue := true
	current := r

	// Go in depth
	for ; index < len(b); index++ {
		// Go into next byte
		if v, ok := current.tree[b[index]]; ok {
			current = v
			if current.last {
				end = index + 1
			}
			continue
		}

		// Continuation not found
		mayContinue = false
		break
	}
	if mayContinue && !current.HasChildren() {
		mayContinue = false
	}
	return end, index, mayContinue, current
}
