package rules

import (
	"slices"

	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// documentParameters lists the parameters of the path items, the operations and
// the components of the document, with references resolved.
func documentParameters(root *yaml.Node) []oas.Parameter {
	return slices.Concat(
		seq.FlatMap(oas.PathItems(root), func(item oas.PathItem) []oas.Parameter {
			return seq.FlatMapPairs(
				oas.Items(oas.Get(item.Node, "parameters")),
				func(i int, entry *yaml.Node) []oas.Parameter {
					return []oas.Parameter{{
						Node: oas.Resolve(root, entry),
						Path: oas.Index(oas.Join(item.Path, "parameters"), i),
					}}
				},
			)
		}),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []oas.Parameter {
			return oas.OwnParameters(root, operation)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "parameters")),
			func(key, value *yaml.Node) []oas.Parameter {
				return []oas.Parameter{{
					Node: oas.Resolve(root, value),
					Path: oas.JoinAll("$.components.parameters", key.Value),
				}}
			},
		),
	)
}
