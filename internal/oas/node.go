package oas

import (
	"iter"
	"strconv"

	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// pairSize is the number of nodes per mapping entry: the key and the value.
const pairSize = 2

// Unwrap removes the document wrapper from n and returns the content node. It
// returns nil for a nil node.
func Unwrap(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}

	return n
}

// IsMap reports whether n is a mapping.
func IsMap(n *yaml.Node) bool {
	return n != nil && n.Kind == yaml.MappingNode
}

// IsSeq reports whether n is a sequence.
func IsSeq(n *yaml.Node) bool {
	return n != nil && n.Kind == yaml.SequenceNode
}

// IsScalar reports whether n is a scalar.
func IsScalar(n *yaml.Node) bool {
	return n != nil && n.Kind == yaml.ScalarNode
}

// Get returns the value of key in the mapping n, or nil when it is absent.
func Get(n *yaml.Node, key string) *yaml.Node {
	_, value := Entry(n, key)

	return value
}

// Entry returns the key node and the value node of key in the mapping n.
func Entry(n *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	if !IsMap(n) {
		return nil, nil
	}

	return entryIn(n.Content, key)
}

func entryIn(content []*yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	for i := range len(content) / pairSize {
		if content[i*pairSize].Value == key {
			return content[i*pairSize], content[i*pairSize+1]
		}
	}

	return nil, nil
}

// Has reports whether the mapping n defines key.
func Has(n *yaml.Node, key string) bool {
	return Get(n, key) != nil
}

// Dig follows keys through nested mappings and returns the node found, or nil.
func Dig(n *yaml.Node, keys ...string) *yaml.Node {
	if len(keys) == 0 {
		return n
	}

	return Dig(Get(n, keys[0]), keys[1:]...)
}

// Pairs iterates over the key and value nodes of the mapping n.
func Pairs(n *yaml.Node) iter.Seq2[*yaml.Node, *yaml.Node] {
	return func(yield func(*yaml.Node, *yaml.Node) bool) {
		if !IsMap(n) {
			return
		}

		for i := range len(n.Content) / pairSize {
			if !yield(n.Content[i*pairSize], n.Content[i*pairSize+1]) {
				return
			}
		}
	}
}

// Items iterates over the elements of the sequence n.
func Items(n *yaml.Node) iter.Seq2[int, *yaml.Node] {
	return func(yield func(int, *yaml.Node) bool) {
		if !IsSeq(n) {
			return
		}

		for i, item := range n.Content {
			if !yield(i, item) {
				return
			}
		}
	}
}

// Text returns the value of the scalar n, or the empty string for any other
// node.
func Text(n *yaml.Node) string {
	if !IsScalar(n) {
		return ""
	}

	return n.Value
}

// StringAt returns the scalar value stored under key in the mapping n.
func StringAt(n *yaml.Node, key string) string {
	return Text(Get(n, key))
}

// IsTrue reports whether n is the boolean scalar true.
func IsTrue(n *yaml.Node) bool {
	return IsScalar(n) && n.Value == "true"
}

// IsNull reports whether n is a YAML null scalar.
func IsNull(n *yaml.Node) bool {
	return IsScalar(n) && (n.Tag == "!!null" || n.Value == "null" || n.Value == "~")
}

// Int returns the integer value of the scalar n and whether it parsed.
func Int(n *yaml.Node) (int, bool) {
	if !IsScalar(n) {
		return 0, false
	}

	value, err := strconv.Atoi(n.Value)

	return value, err == nil
}

// Number returns the numeric value of the scalar n and whether it parsed.
func Number(n *yaml.Node) (float64, bool) {
	if !IsScalar(n) {
		return 0, false
	}

	value, err := strconv.ParseFloat(n.Value, 64)

	return value, err == nil
}

// Strings returns the scalar values of the sequence n.
func Strings(n *yaml.Node) []string {
	return seq.FlatMapPairs(Items(n), func(_ int, item *yaml.Node) []string {
		if IsScalar(item) {
			return []string{item.Value}
		}

		return nil
	})
}

// Keys returns the keys of the mapping n in document order.
func Keys(n *yaml.Node) []string {
	return seq.FlatMapPairs(Pairs(n), func(key, _ *yaml.Node) []string {
		return []string{key.Value}
	})
}

// Empty reports whether n is nil, an empty collection or an empty scalar.
func Empty(n *yaml.Node) bool {
	if n == nil {
		return true
	}

	if IsScalar(n) {
		return n.Value == ""
	}

	return len(n.Content) == 0
}
