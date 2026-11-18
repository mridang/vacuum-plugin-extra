package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// compositionBranchesSameTypeMaxDepth bounds the recursion through nested
// compositions and references so that cyclic schemas terminate.
const compositionBranchesSameTypeMaxDepth = 16

// compositionBranchesSameTypeMinDistinct is the number of distinct types that
// makes a composition inconsistent.
const compositionBranchesSameTypeMinDistinct = 2

// compositionBranchesSameTypeIgnoreNull is the option that skips null leaves
// and drops null from type lists.
const compositionBranchesSameTypeIgnoreNull = "ignoreNull"

// compositionBranchesSameTypeNull is the JSON Schema type that marks a value as
// nullable rather than naming a second type.
const compositionBranchesSameTypeNull = "null"

// CompositionBranchesSameType returns the extra-composition-branches-same-type
// rule.
//
// # What it checks
//
// For every schema that uses allOf, anyOf or oneOf, all the leaf schemas
// reachable through those keywords must declare the same type. Branches are
// followed through $ref and through nested compositions, so only the leaves
// (schemas without a composition keyword of their own) are compared. Leaves
// that declare no type are ignored. A not branch is not a sibling of the
// composition and is never compared.
//
// With ignoreNull (the default) a leaf whose type is null is skipped, and null
// is dropped from a type list, so that [string, "null"] and oneOf [T, null]
// count as a string. null marks nullability in OpenAPI 3.1 rather than a second
// type. Without it, null counts as a type like any other.
//
// A composition that mixes, for example, an object and a string describes a
// value that no consumer can model with one static type, and most code
// generators fail or fall back to an untyped value.
//
// # Where it comes from
//
// The Oxide OpenAPI lint, which requires every leaf of a oneOf, allOf, anyOf or
// not to have the same type. This rule departs from it for not branches, which
// are never compared, and for null leaves, which ignoreNull skips.
//
// # Options
//
//   - ignoreNull: skip null leaves and drop null from type lists before the
//     types are compared. Default: true.
//
// # Bad
//
//	components:
//	  schemas:
//	    Id:
//	      oneOf:
//	        - type: string
//	        - type: integer
//
// # Good
//
//	components:
//	  schemas:
//	    Id:
//	      oneOf:
//	        - type: string
//	          format: uuid
//	        - type: string
//	          format: ulid
//
// # Known false positives
//
// A deliberate union of different primitive types is reported; this is the
// pattern the rule exists to discourage. With ignoreNull set to false, nullable
// unions written as oneOf [T, type null] are reported too. Compositions that
// are only resolved through an external $ref are not followed.
func CompositionBranchesSameType() Rule {
	return New(Spec{
		Name:        "composition-branches-same-type",
		Description: "All leaf schemas of a oneOf, allOf or anyOf must have the same type.",
		HowToFix:    "Make every branch of the composition declare the same type, or restructure the schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        compositionBranchesSameTypeIgnoreNull,
				Description: "Skip null leaves and drop null from type lists before comparing types.",
				Default:     true,
			},
		},
		Check: PerSchema(checkCompositionBranchesSameType),
	})
}

func checkCompositionBranchesSameType(call *Call, schema oas.Schema) []Finding {
	if !compositionBranchesSameTypeHasComposition(schema.Node) {
		return nil
	}

	ignoreNull := call.Bool(compositionBranchesSameTypeIgnoreNull)
	unique := slices.Compact(
		slices.Sorted(slices.Values(compositionBranchesSameTypeCollect(call.Root, schema.Node, 0, ignoreNull))),
	)

	return When(len(unique) >= compositionBranchesSameTypeMinDistinct, func() Finding {
		return Finding{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "all branches of the composition must have the same type, found: " + strings.Join(unique, ", "),
		}
	})
}

func compositionBranchesSameTypeKeywords() []string {
	return []string{compositionBranchesSameTypeAllOf, keywordAnyOf, literalOneof}
}

func compositionBranchesSameTypeHasComposition(schema *yaml.Node) bool {
	return slices.ContainsFunc(compositionBranchesSameTypeKeywords(), func(keyword string) bool {
		return oas.Has(schema, keyword)
	})
}

func compositionBranchesSameTypeCollect(root, schema *yaml.Node, depth int, ignoreNull bool) []string {
	if depth > compositionBranchesSameTypeMaxDepth {
		return nil
	}

	return seq.FlatMapSlice(compositionBranchesSameTypeKeywords(), func(keyword string) []string {
		value := oas.Get(schema, keyword)
		return seq.FlatMapSlice(compositionBranchesSameTypeBranches(value), func(branch *yaml.Node) []string {
			return compositionBranchesSameTypeLeaf(root, oas.Resolve(root, branch), depth+1, ignoreNull)
		})
	})
}

func compositionBranchesSameTypeBranches(value *yaml.Node) []*yaml.Node {
	if oas.IsSeq(value) {
		return value.Content
	}

	return []*yaml.Node{value}
}

func compositionBranchesSameTypeLeaf(root, branch *yaml.Node, depth int, ignoreNull bool) []string {
	if !oas.IsMap(branch) {
		return nil
	}

	if compositionBranchesSameTypeHasComposition(branch) {
		return compositionBranchesSameTypeCollect(root, branch, depth, ignoreNull)
	}

	declared := oas.Types(branch)
	if ignoreNull {
		declared = slices.DeleteFunc(declared, func(name string) bool {
			return name == compositionBranchesSameTypeNull
		})
	}

	if len(declared) == 0 {
		return nil
	}

	return []string{strings.Join(slices.Sorted(slices.Values(declared)), "|")}
}

const (
	compositionBranchesSameTypeAllOf = "allOf"
)
