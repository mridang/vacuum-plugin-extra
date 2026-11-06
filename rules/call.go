package rules

import (
	"maps"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Call is handed to a Check. It carries the matched node, the document root and
// typed accessors for the functionOptions of the rule.
type Call struct {
	// Node is the node matched by the given JSONPath, with any document wrapper
	// removed.
	Node *yaml.Node
	// Root is the root mapping of the document being linted.
	Root *yaml.Node

	options  map[string]any
	defaults map[string]any
}

func newCall(node *yaml.Node, context model.RuleFunctionContext, declared []Option) *Call {
	defaults := maps.Collect(func(yield func(string, any) bool) {
		for _, option := range declared {
			if !yield(option.Name, option.Default) {
				return
			}
		}
	})

	options, _ := context.Options.(map[string]any)

	return &Call{
		Node:     oas.Unwrap(node),
		Root:     rootOf(node, context),
		options:  options,
		defaults: defaults,
	}
}

func rootOf(node *yaml.Node, context model.RuleFunctionContext) *yaml.Node {
	if context.Index != nil {
		if root := oas.Unwrap(context.Index.GetRootNode()); root != nil {
			return root
		}
	}

	return oas.Unwrap(node)
}

func (c *Call) raw(name string) any {
	if value, ok := c.options[name]; ok {
		return value
	}

	return c.defaults[name]
}

// String returns a string option, or the empty string when it is unset.
func (c *Call) String(name string) string {
	value, _ := c.raw(name).(string)

	return value
}

// Bool returns a boolean option, or false when it is unset.
func (c *Call) Bool(name string) bool {
	value, _ := c.raw(name).(bool)

	return value
}

// Int returns an integer option, or zero when it is unset. Numbers decoded from
// YAML may arrive as int, int64 or float64, and all of them are accepted.
func (c *Call) Int(name string) int {
	switch value := c.raw(name).(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

// Has reports whether the option was supplied by the user or has a default.
func (c *Call) Has(name string) bool {
	return c.raw(name) != nil
}

// Strings returns a list option. A single string is treated as a one element
// list, so both "a" and ["a", "b"] are accepted in a ruleset.
func (c *Call) Strings(name string) []string {
	switch value := c.raw(name).(type) {
	case string:
		return []string{value}
	case []string:
		return value
	case []any:
		return stringsFromAny(value)
	default:
		return nil
	}
}

func stringsFromAny(values []any) []string {
	return slices.AppendSeq(
		make([]string, 0, len(values)),
		slices.Values(seq.FlatMapSlice(values, func(value any) []string {
			if text, ok := value.(string); ok {
				return []string{text}
			}

			return nil
		})),
	)
}

// Fold returns the lower-cased, trimmed form of s, the comparison form used for
// case-insensitive matching.
func Fold(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
