package oas

import (
	"strconv"
	"strings"
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
