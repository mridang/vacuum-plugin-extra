package oas

import (
	"strings"

	"github.com/pb33f/go-yaml"
)

// Version returns the value of the top-level openapi (or swagger) keyword.
func Version(root *yaml.Node) string {
	if version := StringAt(root, "openapi"); version != "" {
		return version
	}

	return StringAt(root, "swagger")
}

// IsOAS31 reports whether root is an OpenAPI 3.1 (or later 3.x) document.
func IsOAS31(root *yaml.Node) bool {
	return strings.HasPrefix(Version(root), "3.1") || strings.HasPrefix(Version(root), "3.2")
}

// IsOAS3 reports whether root is any OpenAPI 3.x document.
func IsOAS3(root *yaml.Node) bool {
	return strings.HasPrefix(Version(root), "3.")
}
