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

// hoistRepeatedPathParametersMinOperations is the number of operations that
// must repeat a parameter before it is reported.
const hoistRepeatedPathParametersMinOperations = 2

// HoistRepeatedPathParameters returns the extra-hoist-repeated-path-parameters rule.
//
// # What it checks
//
// When the same path parameter is declared, identically, in the parameters of
// more than one operation of one path item, it should be declared once at the
// path item level instead. Two parameters are identical when their resolved
// objects are structurally equal, regardless of key order. Only parameters with
// in: path are considered, and parameters already declared at the path item
// level are not reported.
//
// # Why
//
// A path parameter belongs to the URL and so to every operation of the path
// item. Repeating it in each operation invites the copies to drift apart and
// makes the description longer than needed.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-avoid-repeating-path-parameters).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    get:
//	      parameters:
//	        - {name: id, in: path, required: true, schema: {type: string}}
//	    delete:
//	      parameters:
//	        - {name: id, in: path, required: true, schema: {type: string}}
//
// # Good
//
//	paths:
//	  /pets/{id}:
//	    parameters:
//	      - {name: id, in: path, required: true, schema: {type: string}}
//	    get: {}
//	    delete: {}
//
// # Known false positives / limitations
//
// Every repeated occurrence is reported, including the first one. A parameter
// that differs between operations, for example in its description, is not
// considered a repeat.
func HoistRepeatedPathParameters() Rule {
	return New(Spec{
		Name:        "hoist-repeated-path-parameters",
		Description: "Identical path parameters repeated across operations must move to the path item.",
		HowToFix:    "Declare the path parameter once in the parameters of the path item and remove it from the operations.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerPathItem(checkHoistRepeatedPathParameters),
	})
}

// hoistRepeatedPathParametersUse is one operation level path parameter.
type hoistRepeatedPathParametersUse struct {
	node      *yaml.Node
	name      string
	path      string
	operation string
	key       string
}

func checkHoistRepeatedPathParameters(call *Call, item oas.PathItem) []Finding {
	uses := seq.FlatMapPairs(oas.Pairs(item.Node), func(method, operation *yaml.Node) []hoistRepeatedPathParametersUse {
		base := oas.Join(oas.Join(item.Path, method.Value), "parameters")

		return seq.FlatMapPairs(
			oas.Items(oas.Get(operation, "parameters")),
			func(i int, entry *yaml.Node) []hoistRepeatedPathParametersUse {
				parameter := oas.Resolve(call.Root, entry)

				return hoistRepeatedPathParametersOnly(
					oas.IsMethod(method.Value) && oas.StringAt(parameter, "in") == literalPath,
					hoistRepeatedPathParametersUse{
						node:      entry,
						name:      oas.StringAt(parameter, "name"),
						path:      oas.Index(base, i),
						operation: method.Value,
						key:       hoistRepeatedPathParametersCanon(call.Root, parameter, 0),
					},
				)
			},
		)
	})

	return hoistRepeatedPathParametersReport(
		seq.GroupBy(slices.Values(uses), func(use hoistRepeatedPathParametersUse) string {
			return use.key
		}),
	)
}

// hoistRepeatedPathParametersReport turns the groups used by more than one
// operation into findings, in document order.
func hoistRepeatedPathParametersReport(groups map[string][]hoistRepeatedPathParametersUse) []Finding {
	findings := seq.FlatMap(maps.Values(groups), func(uses []hoistRepeatedPathParametersUse) []Finding {
		operations := seq.GroupBy(slices.Values(uses), func(use hoistRepeatedPathParametersUse) string {
			return use.operation
		})
		if len(operations) < hoistRepeatedPathParametersMinOperations {
			return nil
		}

		return seq.FlatMapSlice(uses, func(use hoistRepeatedPathParametersUse) []Finding {
			return One(Finding{
				Node:    use.node,
				Path:    use.path,
				Message: "path parameter '" + use.name + "' is repeated in several operations, declare it once on the path item",
			})
		})
	})

	return slices.SortedFunc(slices.Values(findings), func(a, b Finding) int {
		return strings.Compare(a.Path, b.Path)
	})
}

// hoistRepeatedPathParametersOnly returns the single value when condition holds, and nothing otherwise.
func hoistRepeatedPathParametersOnly[T any](condition bool, value T) []T {
	if !condition {
		return nil
	}

	return []T{value}
}

// hoistRepeatedPathParametersMaxDepth bounds the canonical form of a node.
const hoistRepeatedPathParametersMaxDepth = 24

// hoistRepeatedPathParametersCanon returns a canonical string of the node in
// which mapping keys are sorted and local references are resolved.
func hoistRepeatedPathParametersCanon(root, node *yaml.Node, depth int) string {
	resolved := oas.Resolve(root, node)
	if resolved == nil || depth > hoistRepeatedPathParametersMaxDepth {
		return "~"
	}

	switch {
	case oas.IsMap(resolved):
		parts := seq.Map(slices.Values(slices.Sorted(slices.Values(oas.Keys(resolved)))), func(key string) string {
			return key + ":" + hoistRepeatedPathParametersCanon(root, oas.Get(resolved, key), depth+1)
		})

		return "{" + strings.Join(parts, ",") + "}"
	case oas.IsSeq(resolved):
		parts := seq.Map(slices.Values(resolved.Content), func(item *yaml.Node) string {
			return hoistRepeatedPathParametersCanon(root, item, depth+1)
		})

		return "[" + strings.Join(parts, ",") + "]"
	default:
		return resolved.Tag + "=" + resolved.Value
	}
}
