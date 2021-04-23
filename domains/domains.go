package domains

import (
	"fmt"
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

func trimDots(s string) string {
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

type DomainTree interface {
	Put(key string)
	Get(key string) bool
	Dump() string
}

type node struct {
	andChildren bool
	children    map[string]*node
	formatter   func(string) string
}

func New() DomainTree {
	return &node{
		formatter: strings.ToLower,
	}
}

func NewFromList(list []string) DomainTree {
	if list != nil && len(list) > 0 {
		node := New()
		for _, domain := range list {
			node.Put(domain)
		}
		return node
	}
	return nil
}

func (trie *node) Put(key string) {
	key = trimDots(key)
	key = trie.formatter(key)
	currentNode := trie
	for part, i := domainSegmenter(key, 0); part != ""; part, i = domainSegmenter(key, i) {
		if currentNode.andChildren {
			return
		}
		if part == "*" {
			currentNode.andChildren = true
			currentNode.children = nil
			return
		}
		child, _ := currentNode.children[part]
		if child == nil {
			if currentNode.children == nil {
				currentNode.children = map[string]*node{}
			}
			child = &node{}
			currentNode.children[part] = child
		}
		currentNode = child
	}
}

func (trie *node) Get(key string) bool {
	key = trimDots(key)
	currentNode := trie
	for part, i := domainSegmenter(key, 0); part != ""; part, i = domainSegmenter(key, i) {
		if len(part) > 0 && currentNode.andChildren {
			return true
		}
		part = trie.formatter(part)
		currentNode = currentNode.children[part]
		if currentNode == nil {
			return false
		}
	}
	return currentNode.children == nil && !currentNode.andChildren
}

func (trie *node) Dump() string {
	result := ""
	currentNode := trie
	if currentNode.children != nil {
		for name, n := range currentNode.children {
			result += fmt.Sprintf("%s: is wildcard: %v", name, n.andChildren)
			result += fmt.Sprintf("\n %s", n.Dump())
		}
	}
	return result
}
