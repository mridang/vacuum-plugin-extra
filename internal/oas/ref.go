package oas

import (
	"slices"
	"strings"

	"github.com/pb33f/go-yaml"
)

// maxRefDepth bounds $ref chains so that a cycle cannot loop forever.
const maxRefDepth = 32

// Ref returns the value of the $ref keyword of n, or the empty string.
func Ref(n *yaml.Node) string {
	return StringAt(n, "$ref")
}

// Resolve follows local $ref chains starting at n and returns the final target.
// A reference that cannot be resolved, or one that is not local, returns n.
func Resolve(root, n *yaml.Node) *yaml.Node {
	return resolveWithin(root, n, maxRefDepth)
}

func resolveWithin(root, n *yaml.Node, depth int) *yaml.Node {
	ref := Ref(n)
	if depth == 0 || ref == "" {
		return n
	}

	target := Pointer(root, ref)
	if target == nil {
		return n
	}

	return resolveWithin(root, target, depth-1)
}

// Pointer resolves a local JSON pointer such as "#/components/schemas/Pet"
// against root. It returns nil when the pointer is not local or does not exist.
func Pointer(root *yaml.Node, ref string) *yaml.Node {
	pointer, ok := strings.CutPrefix(ref, "#")
	if !ok {
		return nil
	}

	return followTokens(
		Unwrap(root),
		slices.Collect(strings.SplitSeq(strings.TrimPrefix(pointer, "/"), "/")),
	)
}

func followTokens(current *yaml.Node, tokens []string) *yaml.Node {
	if len(tokens) == 0 {
		return current
	}

	if tokens[0] == "" {
		return followTokens(current, tokens[1:])
	}

	next := Get(current, strings.ReplaceAll(strings.ReplaceAll(tokens[0], "~1", "/"), "~0", "~"))
	if next == nil {
		return nil
	}

	return followTokens(next, tokens[1:])
}

// RefName returns the last segment of a $ref, which is the component name for
// references into #/components.
func RefName(ref string) string {
	return ref[strings.LastIndex(ref, "/")+1:]
}
