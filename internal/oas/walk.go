package oas

import (
	"iter"

	"github.com/pb33f/go-yaml"
)

// Node is a node of the document with the JSONPath it was found at.
type Node struct {
	// Key is the mapping key (or "[i]" index) the node was found under.
	Key string
	// Node is the YAML node.
	Node *yaml.Node
	// Path is the JSONPath of the node.
	Path string
}

// Walk iterates over every mapping and sequence value below n, depth first,
// reporting each node with its JSONPath. The start node itself is not reported.
// $ref targets are not followed.
func Walk(n *yaml.Node, path string) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		walk(n, path, yield)
	}
}

func walk(n *yaml.Node, path string, yield func(Node) bool) bool {
	switch {
	case IsMap(n):
		for key, value := range Pairs(n) {
			child := Join(path, key.Value)
			if !yield(Node{Key: key.Value, Node: value, Path: child}) || !walk(value, child, yield) {
				return false
			}
		}
	case IsSeq(n):
		for i, item := range Items(n) {
			child := Index(path, i)
			if !yield(Node{Key: Index("", i), Node: item, Path: child}) || !walk(item, child, yield) {
				return false
			}
		}
	}

	return true
}
