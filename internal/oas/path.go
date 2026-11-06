package oas

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
)

// isIdentifier reports whether key can be written in JSONPath dot notation.
func isIdentifier(key string) bool {
	if key == "" {
		return false
	}

	for i, r := range key {
		letter := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		digit := r >= '0' && r <= '9'
		if !letter && (!digit || i == 0) {
			return false
		}
	}

	return true
}

// Join appends key to the JSONPath base, using dot notation for plain
// identifiers and bracket notation for everything else.
func Join(base string, key string) string {
	if isIdentifier(key) {
		return base + "." + key
	}

	return base + "['" + strings.ReplaceAll(key, "'", `\'`) + "']"
}

// JoinAll appends every key to the JSONPath base in order.
func JoinAll(base string, keys ...string) string {
	if len(keys) == 0 {
		return base
	}

	return JoinAll(Join(base, keys[0]), keys[1:]...)
}

// Index appends an array index to the JSONPath base.
func Index(base string, i int) string {
	return base + "[" + strconv.Itoa(i) + "]"
}

// negationSegments matches the path segments that hold a schema which only
// constrains or negates a value instead of describing it.
var negationSegments = regexp.MustCompile(`\.(not|if|then|else)(\.|\[|$)`)

// propertyNames matches the segments that name a property, so that a property
// called "not" is not mistaken for the keyword.
var propertyNames = regexp.MustCompile(`\.properties(\.[A-Za-z0-9_$-]+|\['[^']*'\])`)

// UnderNegation reports whether the JSONPath of a schema is below a not, if,
// then or else keyword. Such schemas are constraint fragments (for example
// "not: {anyOf: [{type: 'null'}]}"), not descriptions of the data.
func UnderNegation(path string) bool {
	return negationSegments.MatchString(propertyNames.ReplaceAllString(path, ".properties"))
}

var pathToken = regexp.MustCompile(`\.([A-Za-z0-9_$-]+)|\['((?:[^'\\]|\\.)*)'\]|\[(\d+)\]`)

// NodeAt returns the node a JSONPath produced by Join and Index points at, or
// nil when the path does not lead anywhere in root.
func NodeAt(root *yaml.Node, path string) *yaml.Node {
	rest := strings.TrimPrefix(path, "$")
	current := Unwrap(root)

	for rest != "" {
		match := pathToken.FindStringSubmatchIndex(rest)
		if match == nil || match[0] != 0 {
			return nil
		}

		submatch := pathToken.FindStringSubmatch(rest)
		current = step(current, submatch)

		if current == nil {
			return nil
		}

		rest = rest[match[1]:]
	}

	return current
}

func step(node *yaml.Node, token []string) *yaml.Node {
	switch {
	case token[3] != "":
		index, err := strconv.Atoi(token[3])
		if err != nil || node.Kind != yaml.SequenceNode || index >= len(node.Content) {
			return nil
		}

		return node.Content[index]
	case token[2] != "":
		return Get(node, strings.ReplaceAll(token[2], `\'`, "'"))
	default:
		return Get(node, token[1])
	}
}
