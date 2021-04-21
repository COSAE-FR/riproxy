package domains

import (
	"strings"
)

func domainSegmenter(path string, start int) (segment string, next int) {
	if len(path) == 0 || start < 0 || start > len(path)-1 {
		return "", -1
	}
	if start == 0 {
		start = len(path)
	}
	end := strings.LastIndexByte(path[:start], '.')
	if end == -1 {
		return path[:start], -1
	}
	return path[end+1 : start], end
}

type Node struct {
	andChildren bool
	children    map[string]*Node
}

func New() *Node {
	return &Node{}
}

func (trie *Node) Put(key string) {
	node := trie
	for part, i := domainSegmenter(key, 0); part != ""; part, i = domainSegmenter(key, i) {
		if node.andChildren {
			return
		}
		if part == "*" {
			node.andChildren = true
			node.children = nil
			return
		}
		child, _ := node.children[part]
		if child == nil {
			if node.children == nil {
				node.children = map[string]*Node{}
			}
			child = New()
			node.children[part] = child
		}
		node = child
	}
}

func (trie *Node) Get(key string) bool {
	node := trie
	for part, i := domainSegmenter(key, 0); part != ""; part, i = domainSegmenter(key, i) {
		if len(part) > 0 && node.andChildren {
			return true
		}
		node = node.children[part]
		if node == nil {
			return false
		}
	}
	return node.children == nil && !node.andChildren
}
